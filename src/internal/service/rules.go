package service

import (
	"context"
	"errors"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/notify"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// validateRule проверяет правило и подставляет значения по умолчанию.
func validateRule(rule *domain.SavingsRule) error {
	if rule.MinTransfer == 0 {
		rule.MinTransfer = domain.DefaultMinTransfer
	}
	if rule.QuietFrom == "" {
		rule.QuietFrom = notify.DefaultQuietFrom
	}
	if rule.QuietTo == "" {
		rule.QuietTo = notify.DefaultQuietTo
	}
	if rule.Timezone == "" {
		rule.Timezone = notify.DefaultTimezone
	}
	if err := rule.Validate(); err != nil {
		return domain.ValidationError(err.Error())
	}
	_, err := notify.ParseQuiet(rule.QuietFrom, rule.QuietTo, rule.Timezone)
	return err
}

// PutRule создаёт или заменяет правило копилки. Нужна одобренная анкета.
func (s *Service) PutRule(ctx context.Context, rule domain.SavingsRule) (domain.SavingsRule, error) {
	if err := validateRule(&rule); err != nil {
		return rule, err
	}
	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		eligible, income, err := r.OnboardingStatus(ctx, rule.ClientID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrOnboardingRequired
		}
		if err != nil {
			return err
		}
		if !eligible {
			return domain.ErrNotEligible
		}
		if rule.DeclaredIncome == 0 { // доход для холодного старта — из анкеты
			rule.DeclaredIncome = income
		}
		return r.UpsertRule(ctx, rule)
	})
	return rule, err
}

// RulePatch — частичное изменение правила: nil значит «не менять».
type RulePatch struct {
	Paused      *bool   `json:"paused"`
	SavePercent *int    `json:"save_percent"`
	QuietFrom   *string `json:"quiet_from"`
	QuietTo     *string `json:"quiet_to"`
	Timezone    *string `json:"timezone"`
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
		if p.QuietFrom != nil {
			rule.QuietFrom = *p.QuietFrom
		}
		if p.QuietTo != nil {
			rule.QuietTo = *p.QuietTo
		}
		if p.Timezone != nil {
			rule.Timezone = *p.Timezone
		}
		if err := validateRule(&rule); err != nil {
			return err
		}
		return r.UpsertRule(ctx, rule)
	})
	return rule, err
}
