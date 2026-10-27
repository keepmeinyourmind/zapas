package profile

import (
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func TestUpcoming(t *testing.T) {
	recurring := DetectRecurring(anyaSpending(), on(time.October, 20))
	daily := domain.FromRubles(500)

	tests := []struct {
		name         string
		recurring    []domain.RecurringPayment
		declared     domain.Money
		now, until   time.Time
		wantPayments domain.Money
		wantSpending domain.Money
		wantNearest  string // получатель ближайшего платежа, "" — нет
		wantNearAt   time.Time
	}{
		{
			name:      "до следующего дохода: рассрочка 25-го и подписка 3-го",
			recurring: recurring, now: on(time.October, 20), until: on(time.November, 5),
			wantPayments: domain.FromRubles(6000 + 499), wantSpending: domain.FromRubles(500 * 16),
			wantNearest: "Рассрочка за ноутбук", wantNearAt: on(time.October, 25),
		},
		{
			name:      "рассрочка просрочена (должна была 25-го) — деньги на неё всё равно нужны",
			recurring: recurring, now: on(time.October, 27), until: on(time.November, 2),
			wantPayments: domain.FromRubles(6000), wantSpending: domain.FromRubles(500 * 6),
			wantNearest: "Рассрочка за ноутбук", wantNearAt: on(time.October, 27),
		},
		{
			name:      "период два месяца — рассрочка дважды",
			recurring: []domain.RecurringPayment{{Counterparty: "Рассрочка", Amount: domain.FromRubles(6000), DayOfMonth: 25, LastAt: on(time.September, 25)}},
			now:       on(time.October, 20), until: on(time.December, 20),
			wantPayments: domain.FromRubles(12000), wantSpending: domain.FromRubles(500 * 61),
			wantNearest: "Рассрочка", wantNearAt: on(time.October, 25),
		},
		{
			name:     "холодный старт: платежей в истории нет — берём из анкеты пропорционально дням",
			declared: domain.FromRubles(7000), now: on(time.October, 20), until: on(time.November, 4),
			wantPayments: domain.FromRubles(3500), wantSpending: domain.FromRubles(500 * 15),
		},
		{
			name:      "платежи найдены — анкету не учитываем",
			recurring: recurring, declared: domain.FromRubles(100000), now: on(time.October, 11), until: on(time.October, 20),
			wantPayments: 0, wantSpending: domain.FromRubles(500 * 9),
		},
		{
			name: "until не позже now — ничего не нужно", recurring: recurring,
			now: on(time.October, 20), until: on(time.October, 20),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Upcoming(tt.recurring, tt.declared, daily, tt.now, tt.until)
			if got.Payments != tt.wantPayments || got.Spending != tt.wantSpending {
				t.Errorf("платежи %v, траты %v; want %v, %v", got.Payments, got.Spending, tt.wantPayments, tt.wantSpending)
			}
			if got.Total != got.Payments+got.Spending {
				t.Errorf("Total %v ≠ платежи + траты", got.Total)
			}
			switch {
			case tt.wantNearest == "" && got.Nearest != nil:
				t.Errorf("ближайший платёж = %+v, want нет", *got.Nearest)
			case tt.wantNearest != "" && got.Nearest == nil:
				t.Errorf("ближайший платёж не найден, want %s", tt.wantNearest)
			case tt.wantNearest != "" && (got.Nearest.Counterparty != tt.wantNearest || !got.Nearest.At.Equal(tt.wantNearAt)):
				t.Errorf("ближайший = %s %s, want %s %s", got.Nearest.Counterparty, got.Nearest.At.Format("02.01"),
					tt.wantNearest, tt.wantNearAt.Format("02.01"))
			}
		})
	}
}

func TestAddMonthClampsToMonthEnd(t *testing.T) {
	jan31 := time.Date(2026, time.January, 31, 10, 0, 0, 0, time.UTC)
	if got := addMonth(jan31, 31); got.Month() != time.February || got.Day() != 28 {
		t.Errorf("addMonth(31 января) = %s, want 28 февраля", got.Format("02.01"))
	}
	feb28 := time.Date(2026, time.February, 28, 10, 0, 0, 0, time.UTC)
	if got := addMonth(feb28, 31); got.Day() != 31 { // платёж 31-го: в марте снова 31-е
		t.Errorf("addMonth(28 февраля, день 31) = %s, want 31 марта", got.Format("02.01"))
	}
}
