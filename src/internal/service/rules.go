package service

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// PutRule создаёт или заменяет правило копилки. Клиента с таким id создаёт, если его нет.
func (s *Service) PutRule(ctx context.Context, rule domain.SavingsRule) (domain.SavingsRule, error) {
	if rule.MinTransfer == 0 {
		rule.MinTransfer = domain.DefaultMinTransfer
	}
	if err := rule.Validate(); err != nil {
		return rule, domain.ValidationError(err.Error())
	}
	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		if err := r.EnsureClient(ctx, rule.ClientID); err != nil {
			return err
		}
		return r.UpsertRule(ctx, rule)
	})
	return rule, err
}

// RulePatch — частичное изменение правила: nil значит «не менять».
type RulePatch struct {
	Paused      *bool `json:"paused"`
	SavePercent *int  `json:"save_percent"`
}

func (s *Service) PatchRule(ctx context.Context, clientID string, p RulePatch) (domain.SavingsRule, error) {
	var rule domain.SavingsRule
	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		var err error
		if rule, err = r.GetRule(ctx, clientID); err != nil {
			return err
		}
		if p.Paused != nil {
			rule.Paused = *p.Paused
		}
		if p.SavePercent != nil {
			rule.SavePercent = *p.SavePercent
		}
		if err := rule.Validate(); err != nil {
			return domain.ValidationError(err.Error())
		}
		return r.UpsertRule(ctx, rule)
	})
	return rule, err
}
