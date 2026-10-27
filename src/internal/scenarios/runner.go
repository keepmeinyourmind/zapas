package scenarios

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Runner сбрасывает Аню и прогоняет сценарий через тот же конвейер, что и настоящие транзакции.
type Runner struct {
	store *storage.Store
	svc   *service.Service
	clock *clock.Clock
}

func NewRunner(store *storage.Store, svc *service.Service, clk *clock.Clock) *Runner {
	return &Runner{store: store, svc: svc, clock: clk}
}

// Result — итог сценария: решение по ключевому событию (nil — решения нет, например, это не доход).
type Result struct {
	ClientID    string            `json:"client_id"`
	Scenario    string            `json:"scenario"`
	Description string            `json:"description"`
	IncomeKind  domain.IncomeKind `json:"income_kind"`
	Decision    *domain.Decision  `json:"decision"`
	Savings     service.Savings   `json:"savings"`
}

// Run: зафиксировать время → удалить Аню → анкета и правило → история → балансы → событие.
// Время зафиксировано на дате события, поэтому результат всегда одинаковый.
func (r *Runner) Run(ctx context.Context, name string) (Result, error) {
	sc, ok := Get(name)
	if !ok {
		return Result{}, domain.ErrNotFound
	}
	r.clock.Set(EventAt)

	if err := r.store.InTx(ctx, func(repo *storage.Repo) error {
		return repo.DeleteClient(ctx, ClientID)
	}); err != nil {
		return Result{}, err
	}
	if _, err := r.svc.SubmitOnboarding(ctx, sc.Onboarding); err != nil {
		return Result{}, err
	}
	if _, err := r.svc.PutRule(ctx, domain.SavingsRule{
		ClientID: ClientID, GoalName: "Подушка", GoalAmount: domain.FromRubles(36000), TermMonths: 12,
		SavePercent: 10, MinIncome: domain.FromRubles(3000),
	}); err != nil {
		return Result{}, err
	}

	if err := r.store.InTx(ctx, func(repo *storage.Repo) error {
		for _, tx := range sc.History {
			if _, err := repo.InsertTransaction(ctx, tx.Transaction); err != nil {
				return err
			}
			if err := repo.SetIncomeKind(ctx, tx.ID, tx.IncomeKind); err != nil {
				return err
			}
		}
		return repo.SetBalances(ctx, ClientID, sc.Card, sc.Savings)
	}); err != nil {
		return Result{}, err
	}

	event := sc.Event
	event.ClientID = ClientID
	res, _, err := r.svc.ProcessTransaction(ctx, event)
	if err != nil {
		return Result{}, err
	}
	savings, err := r.svc.GetSavings(ctx, ClientID)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ClientID: ClientID, Scenario: sc.Name, Description: sc.Description,
		IncomeKind: res.IncomeKind, Decision: res.Decision, Savings: savings,
	}, nil
}
