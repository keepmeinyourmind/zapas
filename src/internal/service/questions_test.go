package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// decisionStatus — статус решения прямо из базы.
func decisionStatus(t *testing.T, s *Service, id string) domain.DecisionStatus {
	t.Helper()
	list, err := s.ListDecisions(context.Background(), "anya", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		if d.ID == id {
			return d.Status
		}
	}
	t.Fatalf("решение %s не найдено", id)
	return ""
}

// Поздний ответ (таймер ещё не успел закрыть вопрос): вопрос закрывается в базе,
// а следующий из очереди становится активным.
func TestLateAnswerExpiresQuestion(t *testing.T) {
	s, _, clk := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r1, _, err := s.ProcessTransaction(ctx, incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
	if err != nil || r1.Decision == nil {
		t.Fatalf("%v %+v", err, r1.Decision)
	}
	r2, _, err := s.ProcessTransaction(ctx, incoming("t-maria", "Мария", domain.CounterpartyPerson, 2000))
	if err != nil || r2.Decision == nil || r2.Decision.Status != domain.DecisionQueued {
		t.Fatalf("второй вопрос должен встать в очередь: %v %+v", err, r2.Decision)
	}
	clk.Advance(25 * time.Hour)

	if _, err := s.AnswerDecision(ctx, r1.Decision.ID, Answer{Choice: ChoiceYes}); !errors.Is(err, domain.ErrDecisionClosed) {
		t.Errorf("поздний ответ: %v, want ErrDecisionClosed", err)
	}
	if st := decisionStatus(t, s, r1.Decision.ID); st != domain.DecisionExpired {
		t.Errorf("первый вопрос: %s, want expired", st)
	}
	if st := decisionStatus(t, s, r2.Decision.ID); st != domain.DecisionPending {
		t.Errorf("второй вопрос: %s, want pending — он должен стать активным", st)
	}
}

// Объединённый вопрос не предлагает больше безопасной суммы: деньги прошлого поступления
// ещё на карте и уже учтены в новой безопасной сумме.
func TestMergedQuestionNotAboveSafe(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r1, _, err := s.ProcessTransaction(ctx, incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000))
	if err != nil || r1.Decision == nil || r1.Decision.Action != domain.DecisionAsk {
		t.Fatalf("первое поступление — вопрос о сумме: %v %+v", err, r1.Decision)
	}
	// Клиент потратил почти всё с карты.
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		return r.SetBalances(ctx, "anya", 0, 0)
	}); err != nil {
		t.Fatal(err)
	}
	r2, _, err := s.ProcessTransaction(ctx, incoming("t-2", "ООО Ромашка", domain.CounterpartyLegal, 3000))
	if err != nil {
		t.Fatal(err)
	}
	d := r2.Decision
	if d == nil || d.Action != domain.DecisionAsk || d.Explanation == nil {
		t.Fatalf("второе поступление — вопрос с расшифровкой: %+v", d)
	}
	safe := d.Explanation.SafeAmount
	if r1.Decision.Amount+domain.FromRubles(300) <= safe {
		t.Fatalf("тест не проверяет ограничение: сумма %v не больше безопасной %v", r1.Decision.Amount+domain.FromRubles(300), safe)
	}
	if d.Amount != safe {
		t.Errorf("объединённая сумма %v, want безопасную %v", d.Amount, safe)
	}
	seen := map[domain.Money]bool{}
	for _, o := range d.Options {
		if o > safe {
			t.Errorf("вариант %v больше безопасной суммы %v", o, safe)
		}
		if seen[o] {
			t.Errorf("повторяющийся вариант %v в %v", o, d.Options)
		}
		seen[o] = true
	}
}

// «Это ваш доход?» не спрашиваем, если откладывать всё равно не будем.
func TestConfirmIncomeSkippedWhenPaused(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, s *Service, store *storage.Store)
	}{
		{"копилка на паузе", func(t *testing.T, s *Service, _ *storage.Store) {
			putRule(t, s, func(r *domain.SavingsRule) { r.Paused = true })
		}},
		{"цель достигнута", func(t *testing.T, _ *Service, store *storage.Store) {
			ctx := context.Background()
			if err := store.InTx(ctx, func(r *storage.Repo) error {
				return r.SetBalances(ctx, "anya", 0, domain.FromRubles(36000))
			}); err != nil {
				t.Fatal(err)
			}
		}},
		{"меньше min_income", func(t *testing.T, s *Service, _ *storage.Store) {
			putRule(t, s, func(r *domain.SavingsRule) { r.MinIncome = domain.FromRubles(10000) })
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, store, _ := testService(t)
			subscribe(t, s)
			tt.setup(t, s, store)

			res, _, err := s.ProcessTransaction(context.Background(), incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
			if err != nil {
				t.Fatal(err)
			}
			if res.Decision != nil {
				t.Errorf("решения быть не должно, а есть: %+v", res.Decision)
			}
		})
	}
}

// putRule — правило Ани из subscribe с изменением.
func putRule(t *testing.T, s *Service, change func(r *domain.SavingsRule)) {
	t.Helper()
	rule := domain.SavingsRule{
		ClientID: "anya", GoalName: "Подушка", GoalAmount: domain.FromRubles(36000),
		TermMonths: 12, SavePercent: 10,
	}
	change(&rule)
	if _, err := s.PutRule(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
}

// Просроченный вопрос (таймер ещё не закрыл его) не объединяется с новым:
// он закрывается по таймауту, а новый вопрос — только о новом поступлении.
func TestOverdueActiveQuestionIsNotMerged(t *testing.T) {
	s, _, clk := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r1, _, err := s.ProcessTransaction(ctx, incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000))
	if err != nil || r1.Decision == nil || r1.Decision.Action != domain.DecisionAsk {
		t.Fatalf("первое поступление — вопрос о сумме: %v %+v", err, r1.Decision)
	}
	clk.Advance(25 * time.Hour)

	r2, _, err := s.ProcessTransaction(ctx, incoming("t-2", "ООО Ромашка", domain.CounterpartyLegal, 30000))
	if err != nil || r2.Decision == nil {
		t.Fatalf("%v %+v", err, r2.Decision)
	}
	if r2.Decision.Amount != domain.FromRubles(3000) || r2.Decision.Status != domain.DecisionPending {
		t.Errorf("новый вопрос: %v, %s; want 3 000 ₽ и pending без объединения", r2.Decision.Amount, r2.Decision.Status)
	}
	list, err := s.ListDecisions(ctx, "anya", domain.DecisionExpired, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != r1.Decision.ID || !strings.Contains(list[0].Reason, "не ответили") {
		t.Errorf("первый вопрос должен закрыться по таймауту: %+v", list)
	}
}

// Порядок блокировок: везде сначала строка клиента, потом строки решений. Тест держит
// блокировку клиента (как событие банка или новая операция), пока клиент отвечает на вопрос,
// и затем блокирует сам вопрос. При обратном порядке в ответе PostgreSQL ловит взаимную блокировку.
func TestAnswerLocksClientFirst(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	r, _, err := s.ProcessTransaction(ctx, incoming("t-ivan", "Иван Петров", domain.CounterpartyPerson, 5000))
	if err != nil || r.Decision == nil {
		t.Fatalf("%v %+v", err, r.Decision)
	}

	answered := make(chan error, 1)
	err = store.InTx(ctx, func(repo *storage.Repo) error {
		if _, err := repo.GetClientForUpdate(ctx, "anya"); err != nil {
			return err
		}
		go func() {
			_, err := s.AnswerDecision(ctx, r.Decision.ID, Answer{Choice: ChoiceNo})
			answered <- err
		}()
		time.Sleep(300 * time.Millisecond) // ответ успевает дойти до блокировок и ждёт
		_, err := repo.GetDecisionForUpdate(ctx, r.Decision.ID)
		return err
	})
	if err != nil {
		t.Errorf("транзакция с блокировкой клиента: %v", err)
	}
	if err := <-answered; err != nil {
		t.Errorf("ответ клиента: %v", err)
	}
}

// Ответ на вопрос сверяется с безопасной суммой на момент ответа: пока клиент думал,
// он мог потратить деньги, которые понадобятся до следующего дохода.
func TestAnswerRechecksSafeAmount(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	res, _, err := s.ProcessTransaction(ctx, incoming("t-salary", "ООО Ромашка", domain.CounterpartyLegal, 30000))
	if err != nil || res.Decision == nil || res.Decision.Action != domain.DecisionAsk {
		t.Fatalf("новый клиент — вопрос о сумме: %v %+v", err, res.Decision)
	}
	spend := incoming("t-spend", "Магазин", domain.CounterpartyLegal, 28000)
	spend.Type = domain.TxKindOutgoing
	if _, _, err := s.ProcessTransaction(ctx, spend); err != nil {
		t.Fatal(err)
	}

	d, err := s.AnswerDecision(ctx, res.Decision.ID, Answer{Choice: ChoiceAmount, Amount: res.Decision.Amount})
	if err != nil {
		t.Fatal(err)
	}
	if _, savings := balances(t, store); savings != 0 || d.Status != domain.DecisionSkipped {
		t.Errorf("копилка %v, статус %s: на карте осталось 2 000 ₽ при тратах 4 000 ₽ в день — откладывать нельзя", savings, d.Status)
	}
}
