package service

import (
	"context"
	"errors"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// ListDecisions — история решений клиента с причинами, новые первыми.
func (s *Service) ListDecisions(ctx context.Context, clientID string, status domain.DecisionStatus, limit int) ([]domain.Decision, error) {
	repo := s.store.Read()
	if _, err := repo.GetClient(ctx, clientID); err != nil {
		return nil, err
	}
	return repo.ListDecisions(ctx, clientID, status, limit)
}

// Savings — состояние копилки для экрана клиента.
type Savings struct {
	ClientID          string       `json:"client_id"`
	GoalName          string       `json:"goal_name"`
	GoalAmount        domain.Money `json:"goal_amount"`
	Balance           domain.Money `json:"balance"`
	CardBalance       domain.Money `json:"card_balance"`
	ProgressPercent   float64      `json:"progress_percent"` // только для показа, деньги — в копейках
	Paused            bool         `json:"paused"`
	Mode              domain.Mode  `json:"mode"`
	Arrested          bool         `json:"arrested"`
	PendingDecisionID *string      `json:"pending_decision_id"`
}

func (s *Service) GetSavings(ctx context.Context, clientID string) (Savings, error) {
	repo := s.store.Read()
	c, err := repo.GetClient(ctx, clientID)
	if err != nil {
		return Savings{}, err
	}
	rule, err := repo.GetRule(ctx, clientID)
	if err != nil {
		return Savings{}, err
	}
	sv := Savings{
		ClientID: c.ID, GoalName: rule.GoalName, GoalAmount: rule.GoalAmount,
		Balance: c.SavingsBalance, CardBalance: c.CardBalance,
		Paused: rule.Paused, Mode: c.Mode, Arrested: c.Arrested,
	}
	if rule.GoalAmount > 0 {
		sv.ProgressPercent = min(100, float64(c.SavingsBalance)*100/float64(rule.GoalAmount))
	}
	pending, err := repo.ListDecisions(ctx, clientID, domain.DecisionPending, 10)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return Savings{}, err
	}
	for _, d := range pending {
		if isQuestion(d) {
			id := d.ID
			sv.PendingDecisionID = &id
		}
	}
	return sv, nil
}
