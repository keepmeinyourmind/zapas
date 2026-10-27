package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// UpsertOnboarding сохраняет анкету и её итог; повторная анкета заменяет прежнюю.
func (r *Repo) UpsertOnboarding(ctx context.Context, f domain.OnboardingForm, res domain.OnboardingResult, now time.Time) error {
	reasons, err := json.Marshal(res.Reasons)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO onboarding (client_id, data_consent_at, analysis_consent, birth_date, monthly_income,
		                        income_type, income_stability, income_months, has_loans, monthly_payments,
		                        score, eligible, reasons, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $2)
		ON CONFLICT (client_id) DO UPDATE SET
			data_consent_at = EXCLUDED.data_consent_at, analysis_consent = EXCLUDED.analysis_consent,
			birth_date = EXCLUDED.birth_date, monthly_income = EXCLUDED.monthly_income,
			income_type = EXCLUDED.income_type, income_stability = EXCLUDED.income_stability,
			income_months = EXCLUDED.income_months, has_loans = EXCLUDED.has_loans,
			monthly_payments = EXCLUDED.monthly_payments, score = EXCLUDED.score,
			eligible = EXCLUDED.eligible, reasons = EXCLUDED.reasons, updated_at = EXCLUDED.updated_at`,
		f.ClientID, now, f.AnalysisConsent, f.BirthDate, f.MonthlyIncome,
		f.IncomeType, f.IncomeStability, f.IncomeMonths, f.HasLoans, f.MonthlyPayments,
		res.Score, res.Eligible, reasons)
	return err
}

// OnboardingStatus — одобрена ли анкета клиента и какой доход он указал.
// Нет анкеты → domain.ErrNotFound.
func (r *Repo) OnboardingStatus(ctx context.Context, clientID string) (eligible bool, monthlyIncome domain.Money, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT eligible, monthly_income FROM onboarding WHERE client_id = $1`, clientID,
	).Scan(&eligible, &monthlyIncome)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, domain.ErrNotFound
	}
	return eligible, monthlyIncome, err
}

// DeclaredPayments — обязательные платежи в месяц из анкеты (для холодного старта). Нет анкеты — 0.
func (r *Repo) DeclaredPayments(ctx context.Context, clientID string) (domain.Money, error) {
	var p domain.Money
	err := r.db.QueryRow(ctx, `SELECT monthly_payments FROM onboarding WHERE client_id = $1`, clientID).Scan(&p)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return p, err
}
