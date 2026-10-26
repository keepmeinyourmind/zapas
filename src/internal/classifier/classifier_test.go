package classifier

import (
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func TestClassify(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		tx        domain.Transaction
		confirmed map[string]bool
		want      domain.IncomeKind
	}{
		{
			name: "перевод из копилки не доход",
			tx: domain.Transaction{
				Amount: domain.FromRubles(10000), Type: domain.TxKindInternal,
				CounterpartyKind: domain.CounterpartySelf, OccurredAt: now,
			},
			want: domain.IncomeNone,
		},
		{
			name: "возврат не доход",
			tx: domain.Transaction{
				Amount: domain.FromRubles(1500), Type: domain.TxKindRefund,
				Counterparty: "Магазин", CounterpartyKind: domain.CounterpartyLegal, OccurredAt: now,
			},
			want: domain.IncomeNone,
		},
		{
			name: "кредит не доход",
			tx: domain.Transaction{
				Amount: domain.FromRubles(50000), Type: domain.TxKindLoan,
				CounterpartyKind: domain.CounterpartyBank, OccurredAt: now,
			},
			want: domain.IncomeNone,
		},
		{
			name: "первая выплата от организации — зарплата",
			tx: domain.Transaction{
				Amount: domain.FromRubles(30000), Type: domain.TxKindIncoming,
				Counterparty: "ООО Ромашка", CounterpartyKind: domain.CounterpartyLegal, OccurredAt: now,
			},
			want: domain.IncomeSalary,
		},
		{
			name: "перевод со своей карты другого банка не доход",
			tx: domain.Transaction{
				Amount: domain.FromRubles(10000), Type: domain.TxKindIncoming,
				Counterparty: "Моя карта Т-Банка", CounterpartyKind: domain.CounterpartySelf, OccurredAt: now,
			},
			want: domain.IncomeNone,
		},
		{
			name: "перевод от незнакомого физлица не доход",
			tx: domain.Transaction{
				Amount: domain.FromRubles(3000), Type: domain.TxKindIncoming,
				Counterparty: "Иван Петров", CounterpartyKind: domain.CounterpartyPerson, OccurredAt: now,
			},
			want: domain.IncomeNone,
		},
		{
			name: "перевод от подтверждённого заказчика — фриланс",
			tx: domain.Transaction{
				Amount: domain.FromRubles(3000), Type: domain.TxKindIncoming,
				Counterparty: "Иван Петров", CounterpartyKind: domain.CounterpartyPerson, OccurredAt: now,
			},
			confirmed: map[string]bool{"Иван Петров": true},
			want:      domain.IncomeFreelance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.tx, nil, tt.confirmed)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}

// day — дата 2026 года в 10:00 UTC, чтобы история читалась как календарь.
func day(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 10, 0, 0, 0, time.UTC)
}

// payment — входящая выплата от организации.
func payment(counterparty string, rubles int64, at time.Time) domain.Transaction {
	return domain.Transaction{
		Amount: domain.FromRubles(rubles), Type: domain.TxKindIncoming,
		Counterparty: counterparty, CounterpartyKind: domain.CounterpartyLegal, OccurredAt: at,
	}
}

// Выплаты организаций: зарплата, аванс или премия — смотрим на прошлые выплаты
// того же отправителя за последние 40 дней.
func TestClassifyLegalByHistory(t *testing.T) {
	const work = "ООО Ромашка"

	tests := []struct {
		name    string
		history []domain.Transaction
		tx      domain.Transaction
		want    domain.IncomeKind
	}{
		{
			name: "аванс не считается падением дохода",
			history: []domain.Transaction{
				payment(work, 10000, day(time.September, 10)),
				payment(work, 20000, day(time.September, 25)),
			},
			tx:   payment(work, 10000, day(time.October, 10)),
			want: domain.IncomeAdvance,
		},
		{
			name: "зарплата после аванса — зарплата",
			history: []domain.Transaction{
				payment(work, 10000, day(time.September, 10)),
				payment(work, 20000, day(time.September, 25)),
				payment(work, 10000, day(time.October, 10)),
			},
			tx:   payment(work, 20000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
		{
			name: "обычная зарплата при истории",
			history: []domain.Transaction{
				payment(work, 30000, day(time.September, 25)),
			},
			tx:   payment(work, 30000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
		{
			name: "премия — разовая выплата намного больше обычной",
			history: []domain.Transaction{
				payment(work, 30000, day(time.September, 25)),
			},
			tx:   payment(work, 60000, day(time.October, 25)),
			want: domain.IncomeBonus,
		},
		{
			name: "зарплата упала на 20% — всё ещё зарплата, падение оценит движок",
			history: []domain.Transaction{
				payment(work, 30000, day(time.September, 25)),
			},
			tx:   payment(work, 24000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
		{
			name: "выплаты старше 40 дней не учитываются",
			history: []domain.Transaction{
				payment(work, 100000, day(time.August, 1)),
			},
			tx:   payment(work, 30000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
		{
			name: "выплаты другой организации не учитываются",
			history: []domain.Transaction{
				payment("ООО Лютик", 100000, day(time.October, 1)),
			},
			tx:   payment(work, 30000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
		{
			name: "возврат от той же организации не считается выплатой",
			history: []domain.Transaction{
				{
					Amount: domain.FromRubles(100000), Type: domain.TxKindRefund,
					Counterparty: work, CounterpartyKind: domain.CounterpartyLegal, OccurredAt: day(time.October, 1),
				},
			},
			tx:   payment(work, 30000, day(time.October, 25)),
			want: domain.IncomeSalary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.tx, tt.history, nil)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Неизвестный или пустой вид отправителя — безопасный ответ «не доход»,
// чтобы копилка не откладывала с денег, про которые мы ничего не знаем.
func TestUnknownCounterpartyIsNotIncome(t *testing.T) {
	tx := domain.Transaction{
		Amount: domain.FromRubles(30000), Type: domain.TxKindIncoming,
		Counterparty: "Неизвестно", CounterpartyKind: "", OccurredAt: day(time.October, 25),
	}
	if got := Classify(tx, nil, nil); got != domain.IncomeNone {
		t.Errorf("Classify() = %q, want %q", got, domain.IncomeNone)
	}
}
