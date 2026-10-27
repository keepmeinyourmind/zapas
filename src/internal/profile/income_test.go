package profile

import (
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func income(kind domain.IncomeKind, rubles int64, at time.Time) domain.ClassifiedTransaction {
	return domain.ClassifiedTransaction{
		Transaction: domain.Transaction{
			Amount: domain.FromRubles(rubles), Type: domain.TxKindIncoming,
			Counterparty: "ООО Ромашка", CounterpartyKind: domain.CounterpartyLegal, OccurredAt: at,
		},
		IncomeKind: kind,
	}
}

func TestMonthsOfData(t *testing.T) {
	now := on(time.October, 20)
	tests := []struct {
		name    string
		history []domain.ClassifiedTransaction
		want    int
	}{
		{"нет истории", nil, 0},
		{"одна зарплата", []domain.ClassifiedTransaction{income(domain.IncomeSalary, 30000, on(time.October, 5))}, 1},
		{"аванс и зарплата в одном месяце — один месяц", []domain.ClassifiedTransaction{
			income(domain.IncomeAdvance, 10000, on(time.October, 10)),
			income(domain.IncomeSalary, 20000, on(time.October, 25)),
		}, 1},
		{"три месяца подряд", []domain.ClassifiedTransaction{
			income(domain.IncomeSalary, 30000, on(time.August, 25)),
			income(domain.IncomeSalary, 30000, on(time.September, 25)),
			income(domain.IncomeSalary, 30000, on(time.October, 5)),
		}, 3},
		{"не доход не считается", []domain.ClassifiedTransaction{
			income(domain.IncomeNone, 10000, on(time.September, 1)),
			income(domain.IncomeSalary, 30000, on(time.October, 5)),
		}, 1},
		{"старше 6 месяцев не считается", []domain.ClassifiedTransaction{
			income(domain.IncomeSalary, 30000, on(time.January, 25)),
			income(domain.IncomeSalary, 30000, on(time.October, 5)),
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MonthsOfData(tt.history, now); got != tt.want {
				t.Errorf("MonthsOfData() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExpectedIncome(t *testing.T) {
	now := on(time.October, 20)
	history := []domain.ClassifiedTransaction{
		income(domain.IncomeAdvance, 10000, on(time.August, 10)),
		income(domain.IncomeSalary, 20000, on(time.August, 25)),
		income(domain.IncomeAdvance, 12000, on(time.September, 10)),
		income(domain.IncomeSalary, 22000, on(time.September, 25)),
		income(domain.IncomeAdvance, 11000, on(time.October, 10)),
		income(domain.IncomeSalary, 60000, on(time.January, 25)), // старше 6 месяцев
		income(domain.IncomeNone, 5000, on(time.October, 1)),     // не доход
	}
	got := ExpectedIncome(history, "", now)

	want := map[domain.IncomeKind]domain.Money{
		domain.IncomeAdvance: domain.FromRubles(11000), // медиана 10 000, 11 000, 12 000
		domain.IncomeSalary:  domain.FromRubles(21000), // медиана 20 000 и 22 000; январь не считается
	}
	if len(got) != len(want) {
		t.Fatalf("ExpectedIncome() = %v, want %v", got, want)
	}
	for kind, w := range want {
		if got[kind] != w {
			t.Errorf("ExpectedIncome()[%s] = %v, want %v", kind, got[kind], w)
		}
	}
	if _, ok := got[domain.IncomeNone]; ok {
		t.Error("«не доход» не должен попадать в ожидаемый доход")
	}
}

// Поступление, по которому решаем, не должно влиять на «обычный» доход, с которым его сравнивают:
// иначе упавший доход тянет медиану вниз и падение выглядит меньше, чем есть.
func TestExpectedIncomeWithoutCurrent(t *testing.T) {
	now := on(time.October, 20)
	prev := income(domain.IncomeSalary, 30000, on(time.September, 20))
	prev.ID = "sep"
	cur := income(domain.IncomeSalary, 20000, now)
	cur.ID = "oct"
	history := []domain.ClassifiedTransaction{prev, cur}

	t.Run("текущее поступление не входит в медиану", func(t *testing.T) {
		if got := ExpectedIncome(history, "oct", now)[domain.IncomeSalary]; got != domain.FromRubles(30000) {
			t.Errorf("обычная зарплата = %v, want 30 000 ₽ (без текущих 20 000 ₽)", got)
		}
	})
	t.Run("без id текущего — считаем по всей истории", func(t *testing.T) {
		if got := ExpectedIncome(history, "", now)[domain.IncomeSalary]; got != domain.FromRubles(25000) {
			t.Errorf("обычная зарплата = %v, want 25 000 ₽ — медиана 20 000 и 30 000", got)
		}
	})
}
