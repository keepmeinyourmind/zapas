package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Интеграционные тесты с настоящим PostgreSQL. Без TEST_DATABASE_URL пропускаются;
// в CI база поднимается автоматически. Локально:
//
//	docker compose up -d db
//	TEST_DATABASE_URL='postgres://zapas:zapas@localhost:5433/zapas?sslmode=disable' go test ./internal/service/
var testNow = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC) // 15:00 по Москве — не тихие часы

func testService(t *testing.T) (*Service, *storage.Store, *clock.Clock) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан — интеграционный тест пропущен")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := storage.Migrate(pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE clients CASCADE`); err != nil {
		t.Fatal(err)
	}

	clk := clock.New()
	clk.Set(testNow)
	cfg := config.Load()
	store := storage.New(pool)
	return New(store, cfg, clk, slog.New(slog.NewTextHandler(io.Discard, nil))), store, clk
}

// subscribe — Аня прошла анкету и подключила копилку.
func subscribe(t *testing.T, s *Service) {
	t.Helper()
	ctx := context.Background()
	res, err := s.SubmitOnboarding(ctx, domain.OnboardingForm{
		ClientID: "anya", DataConsent: true, AnalysisConsent: true,
		BirthDate:     time.Date(2006, 3, 15, 0, 0, 0, 0, time.UTC),
		MonthlyIncome: domain.FromRubles(30000), IncomeType: domain.IncomeTypeMixed,
		IncomeStability: domain.StabilityIrregular, IncomeMonths: 8,
	})
	if err != nil || !res.Eligible {
		t.Fatalf("анкета: %v %+v", err, res)
	}
	if _, err := s.PutRule(ctx, domain.SavingsRule{
		ClientID: "anya", GoalName: "Подушка", GoalAmount: domain.FromRubles(36000),
		TermMonths: 12, SavePercent: 10,
	}); err != nil {
		t.Fatal(err)
	}
}

func incoming(id, from string, kind domain.CounterpartyKind, rubles int64) domain.Transaction {
	return domain.Transaction{
		ID: id, ClientID: "anya", Amount: domain.FromRubles(rubles), Type: domain.TxKindIncoming,
		Counterparty: from, CounterpartyKind: kind, OccurredAt: testNow,
	}
}

func balances(t *testing.T, store *storage.Store) (card, savings domain.Money) {
	t.Helper()
	c, err := store.Read().GetClient(context.Background(), "anya")
	if err != nil {
		t.Fatal(err)
	}
	return c.CardBalance, c.SavingsBalance
}

// Повторное событие той же транзакции не даёт второго решения и второго перевода.
func TestDuplicateTransactionTransfersOnce(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()
	tx := incoming("t-salary", "ООО Ромашка", domain.CounterpartyLegal, 30000)

	first, created, err := s.ProcessTransaction(ctx, tx)
	if err != nil || !created || first.Decision == nil {
		t.Fatalf("первая обработка: %v, created %v, решение %v", err, created, first.Decision)
	}
	_, savingsAfterFirst := balances(t, store)

	second, created, err := s.ProcessTransaction(ctx, tx)
	if err != nil || created {
		t.Fatalf("повтор: %v, created %v — повтор не должен считаться новой транзакцией", err, created)
	}
	if second.Decision == nil || second.Decision.ID != first.Decision.ID {
		t.Errorf("повтор должен вернуть то же решение: %+v vs %+v", second.Decision, first.Decision)
	}
	card, savings := balances(t, store)
	if savings != savingsAfterFirst {
		t.Errorf("копилка после повтора %v, а после первого раза %v — второй перевод!", savings, savingsAfterFirst)
	}
	if card+savings != domain.FromRubles(30000) {
		t.Errorf("карта %v + копилка %v ≠ 30 000 ₽ — деньги учтены дважды", card, savings)
	}
}

// Перевод из копилки обратно на карту — не доход и не повод откладывать (иначе цикл).
func TestTransferFromSavingsMakesNoDecision(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	tx := incoming("t-back", "Копилка", domain.CounterpartySelf, 10000)
	tx.Type = domain.TxKindInternal

	res, _, err := s.ProcessTransaction(context.Background(), tx)
	if err != nil {
		t.Fatal(err)
	}
	if res.IncomeKind != domain.IncomeNone || res.Decision != nil {
		t.Errorf("перевод из копилки: %s, решение %+v; want none и без решения", res.IncomeKind, res.Decision)
	}
}

// Активный вопрос у клиента один: второй встаёт в очередь и становится активным,
// когда клиент ответит на первый.
func TestQuestionsAreQueued(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r1, _, err := s.ProcessTransaction(ctx, incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
	if err != nil {
		t.Fatal(err)
	}
	r2, _, err := s.ProcessTransaction(ctx, incoming("t-maria", "Мария", domain.CounterpartyPerson, 2000))
	if err != nil {
		t.Fatal(err)
	}
	q1, q2 := r1.Decision, r2.Decision
	if q1 == nil || q2 == nil || q1.Action != domain.DecisionConfirmIncome || q2.Action != domain.DecisionConfirmIncome {
		t.Fatalf("ожидали два вопроса «это ваш доход?»: %+v, %+v", q1, q2)
	}
	if q1.Status != domain.DecisionPending || q2.Status != domain.DecisionQueued {
		t.Fatalf("статусы %s, %s; want pending, queued", q1.Status, q2.Status)
	}
	if q1.ExpiresAt == nil || !q1.ExpiresAt.Equal(testNow.Add(24*time.Hour)) {
		t.Errorf("срок ответа %v, want через 24 часа", q1.ExpiresAt)
	}

	if _, err := s.AnswerDecision(ctx, q2.ID, Answer{Choice: ChoiceNo}); !errors.Is(err, domain.ErrDecisionClosed) {
		t.Errorf("ответ на вопрос из очереди: %v, want ErrDecisionClosed", err)
	}
	if _, err := s.AnswerDecision(ctx, q1.ID, Answer{Choice: ChoiceNo}); err != nil {
		t.Fatal(err)
	}
	pending, err := s.ListDecisions(ctx, "anya", domain.DecisionPending, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != q2.ID {
		t.Errorf("после ответа на первый вопрос активным должен стать второй: %+v", pending)
	}
}

// Ответ после 24 часов отклоняется: вопрос уже закрыт безопасным вариантом.
func TestLateAnswerRejected(t *testing.T) {
	s, _, clk := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r, _, err := s.ProcessTransaction(ctx, incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
	if err != nil || r.Decision == nil {
		t.Fatalf("%v %+v", err, r.Decision)
	}
	clk.Advance(25 * time.Hour)

	expired, err := s.ExpireDue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].Status != domain.DecisionExpired {
		t.Fatalf("через 25 часов вопрос должен закрыться: %+v", expired)
	}
	if _, err := s.AnswerDecision(ctx, r.Decision.ID, Answer{Choice: ChoiceYes}); !errors.Is(err, domain.ErrDecisionClosed) {
		t.Errorf("поздний ответ: %v, want ErrDecisionClosed", err)
	}
}

// Под арестом открытые вопросы закрываются, а ответить на них нельзя.
func TestArrestClosesQuestions(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r, _, err := s.ProcessTransaction(ctx, incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
	if err != nil || r.Decision == nil {
		t.Fatalf("%v %+v", err, r.Decision)
	}
	if _, _, err := s.ProcessAccountEvent(ctx, domain.AccountEvent{
		ID: "ev-1", ClientID: "anya", Type: domain.EventArrestImposed, OccurredAt: testNow,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AnswerDecision(ctx, r.Decision.ID, Answer{Choice: ChoiceYes}); !errors.Is(err, domain.ErrDecisionClosed) {
		t.Errorf("ответ после ареста: %v, want ErrDecisionClosed", err)
	}
}

// Снятие из копилки: деньги переходят на карту один раз, больше, чем есть, снять нельзя,
// под арестом — нельзя совсем.
func TestWithdraw(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		return r.SetBalances(ctx, "anya", domain.FromRubles(1000), domain.FromRubles(5000))
	}); err != nil {
		t.Fatal(err)
	}

	sv, created, err := s.Withdraw(ctx, "anya", "w-1", domain.FromRubles(2000))
	if err != nil || !created || sv.Balance != domain.FromRubles(3000) || sv.CardBalance != domain.FromRubles(3000) {
		t.Fatalf("снятие 2 000 ₽: %v, created %v, копилка %v, карта %v; want 3 000 и 3 000", err, created, sv.Balance, sv.CardBalance)
	}
	sv, created, err = s.Withdraw(ctx, "anya", "w-1", domain.FromRubles(2000))
	if err != nil || created || sv.Balance != domain.FromRubles(3000) {
		t.Errorf("повтор снятия: %v, created %v, копилка %v — повтор не должен снимать второй раз", err, created, sv.Balance)
	}
	var ve domain.ValidationError
	if _, _, err := s.Withdraw(ctx, "anya", "w-2", domain.FromRubles(10000)); !errors.As(err, &ve) {
		t.Errorf("снятие больше, чем в копилке: %v, want ValidationError", err)
	}
	if _, _, err := s.Withdraw(ctx, "anya", "w-3", domain.FromRubles(10000)); !errors.As(err, &ve) {
		t.Errorf("после отказа баланс не должен был измениться: %v", err)
	}

	if _, _, err := s.ProcessAccountEvent(ctx, domain.AccountEvent{
		ID: "ev-1", ClientID: "anya", Type: domain.EventArrestImposed, OccurredAt: testNow,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Withdraw(ctx, "anya", "w-4", domain.FromRubles(1000)); !errors.Is(err, domain.ErrAccountArrested) {
		t.Errorf("снятие под арестом: %v, want ErrAccountArrested", err)
	}
}

// Платёж, который клиент пометил необязательным, больше не резервируется.
func TestExcludedPaymentIsNotReserved(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	// Полгода зарплат 30 000 ₽ 20-го и подписка 2 000 ₽ 22-го — история с высоким доверием.
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		for i, m := 0, time.May; m <= time.September; i, m = i+1, m+1 {
			pay := incoming("salary-"+m.String(), "ООО Ромашка", domain.CounterpartyLegal, 30000)
			pay.OccurredAt = time.Date(2026, m, 20, 12, 0, 0, 0, time.UTC)
			sub := domain.Transaction{
				ID: "sub-" + m.String(), ClientID: "anya", Amount: domain.FromRubles(2000), Type: domain.TxKindOutgoing,
				Counterparty: "Онлайн-кинотеатр", CounterpartyKind: domain.CounterpartyLegal,
				OccurredAt: time.Date(2026, m, 22, 12, 0, 0, 0, time.UTC),
			}
			for _, tx := range []domain.Transaction{pay, sub} {
				if _, err := r.InsertTransaction(ctx, tx); err != nil {
					return err
				}
			}
			if err := r.SetIncomeKind(ctx, pay.ID, domain.IncomeSalary); err != nil {
				return err
			}
		}
		return r.SetBalances(ctx, "anya", domain.FromRubles(5000), 0)
	}); err != nil {
		t.Fatal(err)
	}

	reserved := func(txID string) bool {
		t.Helper()
		res, _, err := s.ProcessTransaction(ctx, incoming(txID, "ООО Ромашка", domain.CounterpartyLegal, 30000))
		if err != nil || res.Decision == nil || res.Decision.Explanation == nil {
			t.Fatalf("решение по зарплате: %v %+v", err, res.Decision)
		}
		for _, it := range res.Decision.Explanation.Upcoming {
			if it.Title == "Онлайн-кинотеатр" {
				return true
			}
		}
		return false
	}

	list, err := s.ListRecurring(ctx, "anya")
	if err != nil || len(list) != 1 || list[0].Counterparty != "Онлайн-кинотеатр" || list[0].Excluded {
		t.Fatalf("регулярные платежи: %v %+v; want кинотеатр, не исключён", err, list)
	}
	if !reserved("salary-oct-1") {
		t.Error("пока платёж обязательный, под него должны резервировать деньги")
	}

	list, err = s.SetPaymentExcluded(ctx, "anya", "Онлайн-кинотеатр", true)
	if err != nil || len(list) != 1 || !list[0].Excluded {
		t.Fatalf("исключение: %v %+v", err, list)
	}
	if reserved("salary-oct-2") {
		t.Error("клиент пометил платёж необязательным — резервировать под него не нужно")
	}
}
