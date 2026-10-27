package service

import (
	"context"
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/profile"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Клиент исключил единственный найденный платёж. Платежи из анкеты — только для холодного старта,
// когда по истории не найдено ни одного платежа; здесь платёж найден, просто необязательный.
// Раньше после исключения в резерв вставали платежи из анкеты, и резерв вырастал.
func TestExcludingLastPaymentDoesNotReserveDeclared(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	// В анкете — обязательные платежи 20 000 ₽ в месяц.
	if _, err := s.SubmitOnboarding(ctx, domain.OnboardingForm{
		ClientID: "anya", DataConsent: true, AnalysisConsent: true,
		BirthDate:     time.Date(2006, 3, 15, 0, 0, 0, 0, time.UTC),
		MonthlyIncome: domain.FromRubles(30000), IncomeType: domain.IncomeTypeMixed,
		IncomeStability: domain.StabilityIrregular, IncomeMonths: 8,
		HasLoans: true, MonthlyPayments: domain.FromRubles(20000),
	}); err != nil {
		t.Fatal(err)
	}

	// Полгода зарплат 30 000 ₽ 20-го и подписка 2 000 ₽ 22-го.
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		for m := time.May; m <= time.September; m++ {
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

	if _, err := s.SetPaymentExcluded(ctx, "anya", "Онлайн-кинотеатр", true); err != nil {
		t.Fatal(err)
	}
	res, _, err := s.ProcessTransaction(ctx, incoming("salary-oct", "ООО Ромашка", domain.CounterpartyLegal, 30000))
	if err != nil || res.Decision == nil || res.Decision.Explanation == nil {
		t.Fatalf("решение по зарплате: %v %+v", err, res.Decision)
	}
	for _, it := range res.Decision.Explanation.Upcoming {
		if it.Title == profile.DeclaredPaymentsTitle || it.Title == "Онлайн-кинотеатр" {
			t.Errorf("в резерве %q на %v — после исключения платежа резервировать нечего", it.Title, it.Amount)
		}
	}
}
