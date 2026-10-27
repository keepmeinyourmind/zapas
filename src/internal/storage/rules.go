package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func (r *Repo) GetRule(ctx context.Context, clientID string) (domain.SavingsRule, error) {
	return r.getRule(ctx, clientID, "")
}

// GetRuleForUpdate — правило с блокировкой строки до конца транзакции: параллельные
// изменения правила идут по очереди и не затирают друг друга.
func (r *Repo) GetRuleForUpdate(ctx context.Context, clientID string) (domain.SavingsRule, error) {
	return r.getRule(ctx, clientID, " FOR UPDATE")
}

func (r *Repo) getRule(ctx context.Context, clientID, lock string) (domain.SavingsRule, error) {
	var rule domain.SavingsRule
	err := r.db.QueryRow(ctx, `
		SELECT client_id, goal_name, goal_amount, term_months, save_percent,
		       min_income, min_transfer, declared_income, paused, quiet_from, quiet_to, timezone
		FROM savings_rules WHERE client_id = $1`+lock, clientID,
	).Scan(&rule.ClientID, &rule.GoalName, &rule.GoalAmount, &rule.TermMonths, &rule.SavePercent,
		&rule.MinIncome, &rule.MinTransfer, &rule.DeclaredIncome, &rule.Paused,
		&rule.QuietFrom, &rule.QuietTo, &rule.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return rule, domain.ErrNotFound
	}
	return rule, err
}

// UpdateRuleAfterOnboarding — после повторной анкеты: новый заявленный доход,
// а если анкета не одобрена — пауза. Правила нет — ничего не делает.
func (r *Repo) UpdateRuleAfterOnboarding(ctx context.Context, clientID string, income domain.Money, eligible bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE savings_rules SET declared_income = $2, paused = paused OR NOT $3, updated_at = now()
		WHERE client_id = $1`, clientID, income, eligible)
	return err
}

// UpsertRule создаёт правило или заменяет существующее.
func (r *Repo) UpsertRule(ctx context.Context, rule domain.SavingsRule) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO savings_rules (client_id, goal_name, goal_amount, term_months, save_percent,
		                           min_income, min_transfer, declared_income, paused,
		                           quiet_from, quiet_to, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (client_id) DO UPDATE SET
			goal_name = EXCLUDED.goal_name, goal_amount = EXCLUDED.goal_amount,
			term_months = EXCLUDED.term_months, save_percent = EXCLUDED.save_percent,
			min_income = EXCLUDED.min_income, min_transfer = EXCLUDED.min_transfer,
			declared_income = EXCLUDED.declared_income, paused = EXCLUDED.paused,
			quiet_from = EXCLUDED.quiet_from, quiet_to = EXCLUDED.quiet_to, timezone = EXCLUDED.timezone,
			updated_at = now()`,
		rule.ClientID, rule.GoalName, rule.GoalAmount, rule.TermMonths, rule.SavePercent,
		rule.MinIncome, rule.MinTransfer, rule.DeclaredIncome, rule.Paused,
		rule.QuietFrom, rule.QuietTo, rule.Timezone)
	return err
}
