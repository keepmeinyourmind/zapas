package scenarios

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Каждый сценарий демо даёт ожидаемое решение — это и есть проверка видео.
// Нужен PostgreSQL (TEST_DATABASE_URL); в CI он поднимается автоматически.
func TestScenarios(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан — интеграционный тест пропущен")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := storage.Migrate(pool); err != nil {
		t.Fatal(err)
	}
	store, clk := storage.New(pool), clock.New()
	svc := service.New(store, config.Load(), clk, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner := NewRunner(store, svc, clk)

	tests := []struct {
		scenario   string
		wantAction domain.DecisionAction // "" — решения нет
		wantAmount domain.Money
		wantReason string
	}{
		{"stable", domain.DecisionAuto, domain.FromRubles(3000), "обычн"},
		{"income_drop", domain.DecisionAsk, domain.FromRubles(1000), "ниже обычного на 20%"},
		{"simple_mode", domain.DecisionAuto, domain.FromRubles(2400), "10%"},
		{"self_transfer", "", 0, ""},
		{"new_client", domain.DecisionAsk, domain.FromRubles(3000), "мы ещё изучаем ваш доход"},
		{"income_rise", domain.DecisionSuggestRaise, domain.FromRubles(3600), "вырос на 20%"},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			for run := 1; run <= 2; run++ { // второй прогон — проверка детерминированности
				res, err := runner.Run(ctx, tt.scenario)
				if err != nil {
					t.Fatal(err)
				}
				d := res.Decision
				switch {
				case tt.wantAction == "" && d != nil:
					t.Fatalf("прогон %d: решения быть не должно, а есть %s %v (%s)", run, d.Action, d.Amount, d.Reason)
				case tt.wantAction == "":
					continue
				case d == nil:
					t.Fatalf("прогон %d: решения нет, want %s %v", run, tt.wantAction, tt.wantAmount)
				case d.Action != tt.wantAction || d.Amount != tt.wantAmount:
					t.Errorf("прогон %d: %s %v, want %s %v (причина: %q)", run, d.Action, d.Amount, tt.wantAction, tt.wantAmount, d.Reason)
				case !strings.Contains(norm(d.Reason), norm(tt.wantReason)):
					t.Errorf("прогон %d: причина %q должна содержать %q", run, d.Reason, tt.wantReason)
				}
			}
		})
	}
}

// norm — для сравнения текстов: без учёта регистра, «е» и «ё» — одна буква.
func norm(s string) string { return strings.ReplaceAll(strings.ToLower(s), "ё", "е") }

// Новая покупка уменьшает доступные деньги, но не повышает прогноз того же предложения.
func TestAnswerAfterSpending(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := storage.Migrate(pool); err != nil {
		t.Fatal(err)
	}
	store, clk := storage.New(pool), clock.New()
	svc := service.New(store, config.Load(), clk, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner := NewRunner(store, svc, clk)
	for _, tt := range []struct {
		name        string
		spend, want int64
		status      domain.DecisionStatus
	}{
		{"расход 500 — осталось безопасно 500", 500, 500, domain.DecisionExecuted},
		{"расход 900 — осталось ровно 100", 900, 100, domain.DecisionExecuted},
		{"расход 950 — меньше минимума", 950, 0, domain.DecisionSkipped},
		{"расход 1000 — свободных денег нет", 1000, 0, domain.DecisionSkipped},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := runner.Run(ctx, "income_drop")
			if err != nil {
				t.Fatal(err)
			}
			if res.Decision == nil || res.Decision.Amount != domain.FromRubles(1000) {
				t.Fatalf("исходное предложение: %+v", res.Decision)
			}
			_, _, err = svc.ProcessTransaction(ctx, domain.Transaction{ID: "expense", ClientID: ClientID, Amount: domain.FromRubles(tt.spend), Type: domain.TxKindOutgoing, Counterparty: "Магазин", CounterpartyKind: domain.CounterpartyLegal, OccurredAt: clk.Now()})
			if err != nil {
				t.Fatal(err)
			}
			d, err := svc.AnswerDecision(ctx, res.Decision.ID, service.Answer{Choice: service.ChoiceAmount, Amount: domain.FromRubles(1000)})
			if err != nil {
				t.Fatal(err)
			}
			sv, err := svc.GetSavings(ctx, ClientID)
			if err != nil {
				t.Fatal(err)
			}
			wantSafe := domain.FromRubles(max(1000-tt.spend-tt.want, 0))
			if sv.SafeAmount == nil || *sv.SafeAmount != wantSafe {
				t.Fatalf("текущая безопасная сумма %v; ожидали %v", sv.SafeAmount, wantSafe)
			}
			if d.Status != tt.status || sv.Balance-res.Savings.Balance != domain.FromRubles(tt.want) {
				t.Fatalf("статус %s, переведено %v; ожидали %s, %d ₽", d.Status, sv.Balance-res.Savings.Balance, tt.status, tt.want)
			}
			// Следующее предложение уже использует обновлённую историю расходов.
			clk.Advance(time.Hour)
			tx := salary("next", 24000)
			tx.ClientID = ClientID
			tx.OccurredAt = clk.Now()
			next, _, err := svc.ProcessTransaction(ctx, tx)
			if err != nil {
				t.Fatal(err)
			}
			if next.Decision == nil || next.Decision.Explanation == nil {
				t.Fatal("нет нового расчёта")
			}
			for _, item := range next.Decision.Explanation.Upcoming {
				if item.Days > 0 && item.PerDay <= domain.FromRubles(600) {
					t.Fatalf("новый прогноз не учёл покупку: %+v", item)
				}
			}
		})
	}
}
