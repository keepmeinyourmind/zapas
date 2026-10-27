package scenarios

import (
	"testing"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/profile"
)

// Опорные числа видео: история Ани даёт высокое доверие, обычный доход 30 000 ₽
// и ровно 25 000 ₽ расходов до следующего дохода. Без базы и без движка.
func TestAnyaNumbers(t *testing.T) {
	s, _ := Get("income_drop")
	history := append(s.History, domain.ClassifiedTransaction{Transaction: s.Event, IncomeKind: domain.IncomeSalary})

	p, recurring := profile.Build(history, s.Onboarding.MonthlyIncome, EventAt)
	if p.Trust != domain.TrustLevelHigh {
		t.Errorf("доверие %d, want High (месяцев с доходом: %d)", p.Trust, p.MonthsOfData)
	}
	if got := p.ExpectedFor(domain.IncomeSalary); got != domain.FromRubles(30000) {
		t.Errorf("обычный доход %v, want 30 000 ₽", got)
	}
	if p.DailySpending != domain.FromRubles(600) {
		t.Errorf("траты в день %v, want 600 ₽", p.DailySpending)
	}
	if len(recurring) != 3 {
		t.Errorf("регулярных платежей %d, want 3 (рассрочка, связь, подписка): %+v", len(recurring), recurring)
	}

	up := profile.Upcoming(recurring, s.Onboarding.MonthlyPayments, p.DailySpending, EventAt, p.NextIncomeAt)
	if up.Total != domain.FromRubles(25000) {
		t.Errorf("до следующего дохода понадобится %v (платежи %v + траты %v), want 25 000 ₽", up.Total, up.Payments, up.Spending)
	}
	if up.Nearest == nil || up.Nearest.Counterparty != rassrochka || up.Nearest.At.Sub(EventAt).Hours() != 5*24 {
		t.Errorf("ближайший платёж %+v, want рассрочка через 5 дней", up.Nearest)
	}

	// Безопасная сумма в income_drop: 2 000 + 24 000 − 25 000 = 1 000 ₽.
	if safe := s.Card + s.Event.Amount - up.Total; safe != domain.FromRubles(1000) {
		t.Errorf("безопасно %v, want 1 000 ₽", safe)
	}
}

func TestScenarioEventsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, name := range Names() {
		s, ok := Get(name)
		if !ok {
			t.Fatalf("сценарий %s не найден", name)
		}
		if prev, dup := seen[s.Event.ID]; dup {
			t.Errorf("id события %s повторяется в %s и %s", s.Event.ID, prev, name)
		}
		seen[s.Event.ID] = name
	}
}
