package profile

import (
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func TestNextIncomeAt(t *testing.T) {
	tests := []struct {
		name    string
		history []domain.ClassifiedTransaction
		now     time.Time
		want    time.Time
	}{
		{"нет истории — через 30 дней", nil, on(time.October, 20), on(time.November, 19)},
		{"только зарплата — через 30 дней после неё", []domain.ClassifiedTransaction{
			income(domain.IncomeSalary, 30000, on(time.October, 5)),
		}, on(time.October, 20), on(time.November, 4)},
		{"после зарплаты следующий доход — аванс", []domain.ClassifiedTransaction{
			income(domain.IncomeAdvance, 10000, on(time.October, 10)),
			income(domain.IncomeSalary, 20000, on(time.October, 25)),
		}, on(time.October, 25), on(time.November, 9)},
		{"не доход не учитывается", []domain.ClassifiedTransaction{
			income(domain.IncomeNone, 5000, on(time.October, 15)),
			income(domain.IncomeSalary, 30000, on(time.October, 5)),
		}, on(time.October, 20), on(time.November, 4)},
		{"выплата давно, ожидаемая дата прошла — через 30 дней от now", []domain.ClassifiedTransaction{
			income(domain.IncomeSalary, 30000, on(time.July, 5)),
		}, on(time.October, 20), on(time.November, 19)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextIncomeAt(tt.history, tt.now); !got.Equal(tt.want) {
				t.Errorf("NextIncomeAt = %s, want %s", got.Format("02.01"), tt.want.Format("02.01"))
			}
		})
	}
}

// Профиль Ани за полгода: высокое доверие, обычная зарплата — медиана, а не анкета.
func TestBuildAnya(t *testing.T) {
	now := on(time.October, 25)
	var history []domain.ClassifiedTransaction
	for m := time.May; m <= time.October; m++ {
		history = append(history, income(domain.IncomeSalary, 30000, on(m, 25)))
	}
	for _, tx := range anyaSpending() {
		history = append(history, domain.ClassifiedTransaction{Transaction: tx, IncomeKind: domain.IncomeNone})
	}

	p, recurring := Build(history, domain.FromRubles(25000), now)

	if p.MonthsOfData != 6 || p.Trust != domain.TrustLevelHigh {
		t.Errorf("месяцев %d, доверие %d; want 6, High", p.MonthsOfData, p.Trust)
	}
	if got := p.ExpectedFor(domain.IncomeSalary); got != domain.FromRubles(30000) {
		t.Errorf("обычная зарплата = %v, want 30 000 ₽ (медиана, анкета при высоком доверии не нужна)", got)
	}
	if !p.NextIncomeAt.Equal(on(time.November, 24)) {
		t.Errorf("следующий доход %s, want 24.11", p.NextIncomeAt.Format("02.01"))
	}
	if len(recurring) != 3 {
		t.Errorf("регулярных платежей %d, want 3: %+v", len(recurring), recurring)
	}
}

// Новый клиент: истории нет — доверие низкое, обычный доход берём из анкеты.
func TestBuildNewClientUsesDeclaredIncome(t *testing.T) {
	now := on(time.October, 25)
	history := []domain.ClassifiedTransaction{income(domain.IncomeSalary, 18000, on(time.October, 25))}

	p, recurring := Build(history, domain.FromRubles(25000), now)

	if p.Trust != domain.TrustLevelLow {
		t.Errorf("доверие %d, want Low", p.Trust)
	}
	if got := p.ExpectedFor(domain.IncomeSalary); got != domain.FromRubles(25000) {
		t.Errorf("обычная зарплата = %v, want 25 000 ₽ из анкеты", got)
	}
	if len(recurring) != 0 {
		t.Errorf("регулярных платежей быть не должно: %+v", recurring)
	}
}
