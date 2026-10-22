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
