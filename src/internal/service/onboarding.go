package service

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/onboarding"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// SubmitOnboarding принимает анкету: проверить → оценить → сохранить итог и режим копилки.
// Клиента создаёт, если его ещё нет. Без согласия на обработку данных анкета не сохраняется.
func (s *Service) SubmitOnboarding(ctx context.Context, f domain.OnboardingForm) (domain.OnboardingResult, error) {
	now := s.now()
	if err := onboarding.Validate(f, now); err != nil {
		return domain.OnboardingResult{}, err
	}
	res := onboarding.Evaluate(f, now, s.onboarding)

	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		if err := r.EnsureClient(ctx, f.ClientID); err != nil {
			return err
		}
		if err := r.UpsertOnboarding(ctx, f, res, now); err != nil {
			return err
		}
		// Копилка уже подключена: обновляем доход для холодного старта, а без одобрения — пауза.
		if _, err := r.GetClientForUpdate(ctx, f.ClientID); err != nil {
			return err
		}
		if err := r.UpdateRuleAfterOnboarding(ctx, f.ClientID, f.MonthlyIncome, res.Eligible); err != nil {
			return err
		}
		if !res.Eligible {
			reason := "Копилка на паузе: по новой анкете откладывать небезопасно — вопрос закрыт"
			if err := s.closeQuestions(ctx, r, f.ClientID, reason); err != nil {
				return err
			}
		}
		return r.SetMode(ctx, f.ClientID, res.Mode)
	})
	return res, err
}
