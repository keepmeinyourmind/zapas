package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// InsertTransaction сохраняет операцию. inserted=false — такая операция уже была (повтор события).
func (r *Repo) InsertTransaction(ctx context.Context, tx domain.Transaction) (inserted bool, err error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO transactions (id, client_id, amount, type, counterparty, counterparty_kind, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		tx.ID, tx.ClientID, tx.Amount, tx.Type, tx.Counterparty, tx.CounterpartyKind, tx.OccurredAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) SetIncomeKind(ctx context.Context, txID string, kind domain.IncomeKind) error {
	_, err := r.db.Exec(ctx, `UPDATE transactions SET income_kind = $2 WHERE id = $1`, txID, kind)
	return err
}

// GetIncomeKind возвращает метку классификатора уже сохранённой операции.
func (r *Repo) GetIncomeKind(ctx context.Context, txID string) (domain.IncomeKind, error) {
	var kind *string
	err := r.db.QueryRow(ctx, `SELECT income_kind FROM transactions WHERE id = $1`, txID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if kind == nil {
		return domain.IncomeNone, nil
	}
	return domain.IncomeKind(*kind), nil
}

// ListClassified — операции с меткой классификатора, новые первыми.
func (r *Repo) ListClassified(ctx context.Context, clientID string, limit int) ([]domain.ClassifiedTransaction, error) {
	return r.queryClassified(ctx, `SELECT `+classifiedColumns+` FROM transactions WHERE client_id = $1
		ORDER BY occurred_at DESC, id DESC LIMIT $2`, clientID, limit)
}

// ConfirmedSenders — отправители-физлица, которых клиент подтвердил как источник дохода.
func (r *Repo) ConfirmedSenders(ctx context.Context, clientID string) (map[string]bool, error) {
	return r.queryStringSet(ctx, `SELECT counterparty FROM confirmed_senders WHERE client_id = $1`, clientID)
}

// ListIncomeHistory — операции клиента с меткой классификатора с момента since, от старых к новым.
func (r *Repo) ListIncomeHistory(ctx context.Context, clientID string, since time.Time) ([]domain.ClassifiedTransaction, error) {
	return r.queryClassified(ctx, `SELECT `+classifiedColumns+` FROM transactions
		WHERE client_id = $1 AND occurred_at >= $2 ORDER BY occurred_at, id`, clientID, since)
}

// classifiedColumns — колонки операции с меткой классификатора, в порядке queryClassified.
const classifiedColumns = `id, client_id, amount, type, counterparty, counterparty_kind, occurred_at,
	COALESCE(income_kind, 'none')`

// queryClassified выполняет запрос по classifiedColumns и читает все строки.
func (r *Repo) queryClassified(ctx context.Context, sql string, args ...any) ([]domain.ClassifiedTransaction, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ClassifiedTransaction{}
	for rows.Next() {
		var t domain.ClassifiedTransaction
		if err := rows.Scan(&t.ID, &t.ClientID, &t.Amount, &t.Type, &t.Counterparty,
			&t.CounterpartyKind, &t.OccurredAt, &t.IncomeKind); err != nil {
			return nil, err
		}
		t.OccurredAt = t.OccurredAt.UTC() // pgx отдаёт время в поясе компьютера; держим всё в UTC
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddConfirmedSender — клиент подтвердил, что переводы от этого физлица — его доход.
func (r *Repo) AddConfirmedSender(ctx context.Context, clientID, counterparty string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO confirmed_senders (client_id, counterparty) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, clientID, counterparty)
	return err
}

// GetTransaction — операция по id.
func (r *Repo) GetTransaction(ctx context.Context, id string) (domain.Transaction, error) {
	var t domain.Transaction
	err := r.db.QueryRow(ctx, `
		SELECT id, client_id, amount, type, counterparty, counterparty_kind, occurred_at
		FROM transactions WHERE id = $1`, id,
	).Scan(&t.ID, &t.ClientID, &t.Amount, &t.Type, &t.Counterparty, &t.CounterpartyKind, &t.OccurredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, domain.ErrNotFound
	}
	t.OccurredAt = t.OccurredAt.UTC()
	return t, err
}
