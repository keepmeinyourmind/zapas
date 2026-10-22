package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// EnsureClient создаёт клиента, если его ещё нет (имя = id). В MVP авторизации нет.
func (r *Repo) EnsureClient(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO clients (id, name) VALUES ($1, $1) ON CONFLICT (id) DO NOTHING`, id)
	return err
}

func (r *Repo) GetClient(ctx context.Context, id string) (domain.Client, error) {
	var c domain.Client
	err := r.db.QueryRow(ctx,
		`SELECT id, name, card_balance, credit_limit, savings_balance FROM clients WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.CardBalance, &c.CreditLimit, &c.SavingsBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, domain.ErrNotFound
	}
	return c, err
}
