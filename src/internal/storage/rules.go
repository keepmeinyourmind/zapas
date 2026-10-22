package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func (r *Repo) GetRule(ctx context.Context, clientID string) (domain.SavingsRule, error) {
	var rule domain.SavingsRule
	err := r.db.QueryRow(ctx, `
		SELECT client_id, goal_name, goal_amount, term_months, save_percent,
		       min_income, min_transfer, declared_income, paused
		FROM savings_rules WHERE client_id = $1`, clientID,
	).Scan(&rule.ClientID, &rule.GoalName, &rule.GoalAmount, &rule.TermMonths, &rule.SavePercent,
		&rule.MinIncome, &rule.MinTransfer, &rule.DeclaredIncome, &rule.Paused)
	if errors.Is(err, pgx.ErrNoRows) {
		return rule, domain.ErrNotFound
	}
	return rule, err
}

// UpsertRule создаёт правило или заменяет существующее.
func (r *Repo) UpsertRule(ctx context.Context, rule domain.SavingsRule) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO savings_rules (client_id, goal_name, goal_amount, term_months, save_percent,
		                           min_income, min_transfer, declared_income, paused)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (client_id) DO UPDATE SET
			goal_name = EXCLUDED.goal_name, goal_amount = EXCLUDED.goal_amount,
			term_months = EXCLUDED.term_months, save_percent = EXCLUDED.save_percent,
			min_income = EXCLUDED.min_income, min_transfer = EXCLUDED.min_transfer,
			declared_income = EXCLUDED.declared_income, paused = EXCLUDED.paused,
			updated_at = now()`,
		rule.ClientID, rule.GoalName, rule.GoalAmount, rule.TermMonths, rule.SavePercent,
		rule.MinIncome, rule.MinTransfer, rule.DeclaredIncome, rule.Paused)
	return err
}
