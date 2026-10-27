package service

import (
	"context"
	"fmt"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/profile"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Withdraw снимает деньги из копилки на карту. Идемпотентно по id: повторный запрос
// с тем же id ничего не меняет. created=false — это повтор.
func (s *Service) Withdraw(ctx context.Context, clientID, id string, amount domain.Money) (sv Savings, created bool, err error) {
	switch {
	case id == "":
		return sv, false, domain.ValidationError("Не указан id снятия")
	case amount <= 0:
		return sv, false, domain.ValidationError("Сумма должна быть больше нуля")
	}
	err = s.store.InTx(ctx, func(r *storage.Repo) error {
		c, err := r.GetClientForUpdate(ctx, clientID)
		if err != nil {
			return err
		}
		if c.Arrested {
			return domain.ErrAccountArrested
		}
		inserted, err := r.InsertWithdrawal(ctx, id, clientID, amount, s.now())
		if err != nil || !inserted {
			return err // повтор: деньги второй раз не двигаем
		}
		if amount > c.SavingsBalance {
			return domain.ValidationError(fmt.Sprintf("В копилке только %s", c.SavingsBalance))
		}
		created = true
		return r.MoveToCard(ctx, clientID, amount)
	})
	if err != nil {
		return sv, false, err
	}
	sv, err = s.GetSavings(ctx, clientID)
	return sv, created, err
}

// RecurringPayment — регулярный платёж для экрана «Обязательные платежи».
type RecurringPayment struct {
	Counterparty string       `json:"counterparty"`
	Amount       domain.Money `json:"amount"`
	DayOfMonth   int          `json:"day_of_month"`
	LastAt       time.Time    `json:"last_at"`
	Excluded     bool         `json:"excluded"` // клиент сказал «необязательный» — деньги под него не резервируем
}

// ListRecurring — регулярные платежи, найденные по истории, с пометкой «исключён клиентом».
func (s *Service) ListRecurring(ctx context.Context, clientID string) ([]RecurringPayment, error) {
	repo := s.store.Read()
	if _, err := repo.GetClient(ctx, clientID); err != nil {
		return nil, err
	}
	now := s.now()
	history, err := repo.ListIncomeHistory(ctx, clientID, now.Add(-profileWindow))
	if err != nil {
		return nil, err
	}
	excluded, err := repo.ExcludedPayments(ctx, clientID)
	if err != nil {
		return nil, err
	}
	_, recurring := profile.Build(history, 0, now)

	out := []RecurringPayment{}
	for _, p := range recurring {
		out = append(out, RecurringPayment{
			Counterparty: p.Counterparty, Amount: p.Amount, DayOfMonth: p.DayOfMonth,
			LastAt: p.LastAt, Excluded: excluded[p.Counterparty],
		})
	}
	return out, nil
}

// SetPaymentExcluded — клиент помечает платёж необязательным или возвращает его в обязательные.
func (s *Service) SetPaymentExcluded(ctx context.Context, clientID, counterparty string, excluded bool) ([]RecurringPayment, error) {
	if counterparty == "" {
		return nil, domain.ValidationError("Не указан получатель платежа")
	}
	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		if _, err := r.GetClient(ctx, clientID); err != nil {
			return err
		}
		return r.SetPaymentExcluded(ctx, clientID, counterparty, excluded)
	})
	if err != nil {
		return nil, err
	}
	return s.ListRecurring(ctx, clientID)
}

// withoutExcluded — регулярные платежи без тех, что клиент пометил необязательными.
func withoutExcluded(recurring []domain.RecurringPayment, excluded map[string]bool) []domain.RecurringPayment {
	var out []domain.RecurringPayment
	for _, p := range recurring {
		if !excluded[p.Counterparty] {
			out = append(out, p)
		}
	}
	return out
}
