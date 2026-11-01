package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Operation — единая запись о движении средств в/из копилки.
type Operation struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"` // "transfer_in" или "withdrawal"
	Amount     domain.Money `json:"amount"`
	CreatedAt  time.Time    `json:"created_at"`
	DecisionID *string      `json:"decision_id,omitempty"` // nil для ручных снятий
}

// ListOperations возвращает хронологию всех движений копилки, новые первыми.
func (r *Repo) ListOperations(ctx context.Context, clientID string, limit int) ([]Operation, error) {
	const q = `
		SELECT id::text AS id, 'transfer_in' AS type, amount, created_at, decision_id::text
		FROM transfers WHERE client_id = $1
		UNION ALL
		SELECT id, 'withdrawal' AS type, amount, created_at, NULL::text
		FROM withdrawals WHERE client_id = $1
		UNION ALL
		SELECT id, 'transfer_in' AS type, amount, created_at, NULL::text
		FROM deposits WHERE client_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, clientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []Operation
	for rows.Next() {
		var op Operation
		var decID sql.NullString // безопасно читаем nullable текст

		if err := rows.Scan(&op.ID, &op.Type, &op.Amount, &op.CreatedAt, &decID); err != nil {
			return nil, err
		}

		if decID.Valid {
			op.DecisionID = &decID.String
		}
		op.CreatedAt = op.CreatedAt.UTC()
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if ops == nil {
		ops = []Operation{} // фронтенд получает [], а не null
	}
	return ops, nil
}
