package scenarios

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

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
