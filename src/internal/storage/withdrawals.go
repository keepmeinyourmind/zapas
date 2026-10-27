package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// InsertWithdrawal записывает снятие из копилки. inserted=false — снятие с таким id уже было.
func (r *Repo) InsertWithdrawal(ctx context.Context, id, clientID string, amount domain.Money, at time.Time) (inserted bool, err error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO withdrawals (id, client_id, amount, created_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`, id, clientID, amount, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// GetWithdrawal — кто и сколько снимал по этому id. Нет снятия → domain.ErrNotFound.
func (r *Repo) GetWithdrawal(ctx context.Context, id string) (clientID string, amount domain.Money, err error) {
	err = r.db.QueryRow(ctx, `SELECT client_id, amount FROM withdrawals WHERE id = $1`, id).Scan(&clientID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, domain.ErrNotFound
	}
	return clientID, amount, err
}

// MoveToCard переводит сумму из копилки на карту.
func (r *Repo) MoveToCard(ctx context.Context, clientID string, amount domain.Money) error {
	_, err := r.db.Exec(ctx, `
		UPDATE clients SET savings_balance = savings_balance - $2, card_balance = card_balance + $2
		WHERE id = $1`, clientID, amount)
	return err
}

// ExcludedPayments — платежи, которые клиент пометил как необязательные.
func (r *Repo) ExcludedPayments(ctx context.Context, clientID string) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT counterparty FROM recurring_exclusions WHERE client_id = $1`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// SetPaymentExcluded помечает платёж необязательным (excluded=true) или снимает пометку.
func (r *Repo) SetPaymentExcluded(ctx context.Context, clientID, counterparty string, excluded bool) error {
	if excluded {
		_, err := r.db.Exec(ctx, `INSERT INTO recurring_exclusions (client_id, counterparty) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, clientID, counterparty)
		return err
	}
	_, err := r.db.Exec(ctx, `DELETE FROM recurring_exclusions WHERE client_id = $1 AND counterparty = $2`, clientID, counterparty)
	return err
}
