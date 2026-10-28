package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// addClient — второй клиент Борис с деньгами на карте и в копилке.
func addClient(t *testing.T, store *storage.Store, id string) {
	t.Helper()
	ctx := context.Background()
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		if err := r.EnsureClient(ctx, id); err != nil {
			return err
		}
		return r.SetBalances(ctx, id, domain.FromRubles(1000), domain.FromRubles(5000))
	}); err != nil {
		t.Fatal(err)
	}
}

// Неизвестный тип операции или вид отправителя — ошибка данных (400), а не «плюс на карту».
func TestUnknownTxTypeOrCounterpartyIsValidationError(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	cases := []struct {
		name   string
		mutate func(tx *domain.Transaction)
	}{
		{"неизвестный тип", func(tx *domain.Transaction) { tx.Type = "gift" }},
		{"неизвестный вид отправителя", func(tx *domain.Transaction) { tx.CounterpartyKind = "alien" }},
		{"пустой вид отправителя", func(tx *domain.Transaction) { tx.CounterpartyKind = "" }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tx := incoming("t-bad", "ООО Ромашка", domain.CounterpartyLegal, 1000)
			tt.mutate(&tx)
			var ve domain.ValidationError
			if _, _, err := s.ProcessTransaction(ctx, tx); !errors.As(err, &ve) {
				t.Errorf("%s: %v, want ValidationError", tt.name, err)
			}
		})
	}
}

// Кредит — не свои деньги клиента: баланс карты (свои деньги) от него не растёт.
func TestLoanKeepsOwnCardBalance(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	before, _ := balances(t, store)

	loan := incoming("t-loan", "Банк", domain.CounterpartyBank, 50000)
	loan.Type = domain.TxKindLoan
	if _, _, err := s.ProcessTransaction(context.Background(), loan); err != nil {
		t.Fatal(err)
	}
	if after, _ := balances(t, store); after != before {
		t.Errorf("карта после кредита %v, want %v — кредит не свои деньги", after, before)
	}
}

// Тот же id, но другие данные — это не повтор, а конфликт: чужое решение не отдаём.
func TestSameTxIDWithOtherDataIsConflict(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	addClient(t, store, "boris")
	ctx := context.Background()

	if _, _, err := s.ProcessTransaction(ctx, incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000)); err != nil {
		t.Fatal(err)
	}
	foreign := incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000)
	foreign.ClientID = "boris"
	otherAmount := incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 31000)
	otherType := incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000)
	otherType.Type = domain.TxKindRefund
	otherSender := incoming("t-1", "ООО Лютик", domain.CounterpartyLegal, 30000)
	otherKind := incoming("t-1", "ООО Ромашка", domain.CounterpartyPerson, 30000)
	otherTime := incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000)
	otherTime.OccurredAt = otherTime.OccurredAt.Add(time.Hour)

	cases := []struct {
		name string
		tx   domain.Transaction
	}{
		{"другой клиент", foreign},
		{"другая сумма", otherAmount},
		{"другой тип", otherType},
		{"другой отправитель", otherSender},
		{"другой вид отправителя", otherKind},
		{"другое время", otherTime},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := s.ProcessTransaction(ctx, tt.tx)
			if !errors.Is(err, domain.ErrIDConflict) {
				t.Errorf("%s: %v (решение %+v), want ErrIDConflict", tt.name, err, res.Decision)
			}
		})
	}
}

// Снятие с тем же id: чужой клиент или другая сумма — конфликт; повтор после ареста — просто повтор.
func TestWithdrawIDConflictAndRetryAfterArrest(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	addClient(t, store, "boris")
	ctx := context.Background()
	if err := store.InTx(ctx, func(r *storage.Repo) error {
		return r.SetBalances(ctx, "anya", domain.FromRubles(1000), domain.FromRubles(5000))
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Withdraw(ctx, "anya", "w-1", domain.FromRubles(2000)); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		clientID string
		amount   domain.Money
	}{
		{"чужой id", "boris", domain.FromRubles(2000)},
		{"другая сумма", "anya", domain.FromRubles(3000)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := s.Withdraw(ctx, tt.clientID, "w-1", tt.amount); !errors.Is(err, domain.ErrIDConflict) {
				t.Errorf("%s: %v, want ErrIDConflict", tt.name, err)
			}
		})
	}

	if _, _, err := s.ProcessAccountEvent(ctx, domain.AccountEvent{
		ID: "ev-1", ClientID: "anya", Type: domain.EventArrestImposed, OccurredAt: testNow,
	}); err != nil {
		t.Fatal(err)
	}
	sv, created, err := s.Withdraw(ctx, "anya", "w-1", domain.FromRubles(2000))
	if err != nil || created || sv.Balance != domain.FromRubles(3000) {
		t.Errorf("повтор снятия после ареста: %v, created %v, копилка %v; want повтор без ошибки", err, created, sv.Balance)
	}
}

// Минимальный перевод меньше 100 ₽ не принимаем.
func TestRuleMinTransferBelowDefault(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	var ve domain.ValidationError
	_, err := s.PutRule(context.Background(), domain.SavingsRule{
		ClientID: "anya", GoalName: "Подушка", GoalAmount: domain.FromRubles(36000),
		TermMonths: 12, SavePercent: 10, MinTransfer: 1,
	})
	if !errors.As(err, &ve) {
		t.Errorf("min_transfer 1 коп.: %v, want ValidationError", err)
	}
}

// Повторная анкета без одобрения ставит копилку на паузу, а снять паузу нельзя.
func TestIneligibleReonboardingPausesRule(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	res, err := s.SubmitOnboarding(ctx, domain.OnboardingForm{
		ClientID: "anya", DataConsent: true, AnalysisConsent: true,
		BirthDate:     time.Date(2006, 3, 15, 0, 0, 0, 0, time.UTC),
		MonthlyIncome: domain.FromRubles(20000), IncomeType: domain.IncomeTypeMixed,
		IncomeStability: domain.StabilityIrregular, IncomeMonths: 8,
		MonthlyPayments: domain.FromRubles(20000),
	})
	if err != nil || res.Eligible {
		t.Fatalf("анкета: %v %+v, want не одобрена", err, res)
	}
	rule, err := store.Read().GetRule(ctx, "anya")
	if err != nil {
		t.Fatal(err)
	}
	if !rule.Paused || rule.DeclaredIncome != domain.FromRubles(20000) {
		t.Errorf("правило после анкеты: paused %v, доход %v; want пауза и 20 000 ₽", rule.Paused, rule.DeclaredIncome)
	}

	unpause := false
	if _, err := s.PatchRule(ctx, "anya", RulePatch{Paused: &unpause}); !errors.Is(err, domain.ErrNotEligible) {
		t.Errorf("снятие паузы: %v, want ErrNotEligible", err)
	}
}

// Как операция меняет свои и кредитные деньги на карте. Траты сначала идут из своих денег:
// так безопасная сумма считается осторожно, а свои деньги не уходят в минус из-за трат кредита.
func TestApplyToCard(t *testing.T) {
	rub := domain.FromRubles
	tests := []struct {
		name              string
		own, loan         domain.Money
		tx                domain.Transaction
		wantOwn, wantLoan domain.Money
	}{
		{"поступление — свои деньги", rub(1000), 0, tx(domain.TxKindIncoming, 5000), rub(6000), 0},
		{"кредит — не свои деньги", rub(1000), 0, tx(domain.TxKindLoan, 50000), rub(1000), rub(50000)},
		{"трата из своих денег", rub(6000), rub(50000), tx(domain.TxKindOutgoing, 5000), rub(1000), rub(50000)},
		{"трата больше своих — остаток из кредита", rub(1000), rub(50000), tx(domain.TxKindOutgoing, 48000), 0, rub(3000)},
		{"денег не хватает совсем — свои в минус", rub(1000), rub(2000), tx(domain.TxKindOutgoing, 5000), rub(-2000), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			own, loan := applyToCard(tt.own, tt.loan, tt.tx)
			if own != tt.wantOwn || loan != tt.wantLoan {
				t.Errorf("свои %v, кредит %v; want %v, %v", own, loan, tt.wantOwn, tt.wantLoan)
			}
		})
	}
}

func tx(kind domain.TxKind, rubles int64) domain.Transaction {
	return domain.Transaction{Type: kind, Amount: domain.FromRubles(rubles)}
}

// Кредит, а потом покупка на эти деньги: свои деньги на карте не уходят в минус.
func TestSpendingLoanMoneyKeepsOwnBalance(t *testing.T) {
	s, store, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()
	loan := incoming("t-loan", "ВТБ", domain.CounterpartyBank, 50000)
	loan.Type = domain.TxKindLoan
	buy := incoming("t-laptop", "DNS", domain.CounterpartyLegal, 48000)
	buy.Type = domain.TxKindOutgoing
	for _, op := range []domain.Transaction{loan, buy} {
		if _, _, err := s.ProcessTransaction(ctx, op); err != nil {
			t.Fatal(err)
		}
	}
	if card, _ := balances(t, store); card != 0 {
		t.Errorf("свои деньги на карте %v, want 0: покупка оплачена кредитом", card)
	}
}

// PostgreSQL хранит время с точностью до микросекунды: повтор с наносекундами — тот же повтор.
func TestRetryWithNanosecondsIsNotConflict(t *testing.T) {
	s, _, _ := testService(t)
	subscribe(t, s)
	ctx := context.Background()

	tx := incoming("t-1", "ООО Ромашка", domain.CounterpartyLegal, 30000)
	tx.OccurredAt = tx.OccurredAt.Add(123456789 * time.Nanosecond)
	if _, _, err := s.ProcessTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ProcessTransaction(ctx, tx); err != nil {
		t.Errorf("повтор той же операции: %v, want без ошибки", err)
	}
}
