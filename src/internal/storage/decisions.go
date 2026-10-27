package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

const decisionColumns = `id, client_id, source_tx_id, action, amount, options, reason, status,
	created_at, expires_at, notify_at, explanation`

func scanDecision(row pgx.Row) (domain.Decision, error) {
	var d domain.Decision
	var options, explanation []byte
	err := row.Scan(&d.ID, &d.ClientID, &d.SourceTxID, &d.Action, &d.Amount, &options, &d.Reason, &d.Status,
		&d.CreatedAt, &d.ExpiresAt, &d.NotifyAt, &explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, domain.ErrNotFound
	}
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(options, &d.Options); err != nil {
		return d, err
	}
	if explanation != nil {
		if err := json.Unmarshal(explanation, &d.Explanation); err != nil {
			return d, err
		}
	}
	// pgx отдаёт время в поясе компьютера; держим всё в UTC.
	d.CreatedAt = d.CreatedAt.UTC()
	d.ExpiresAt = utcPtr(d.ExpiresAt)
	d.NotifyAt = utcPtr(d.NotifyAt)
	return d, nil
}

// marshalExplanation — расшифровка в JSONB; nil — NULL в базе.
func marshalExplanation(e *domain.Explanation) ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	return json.Marshal(e)
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func (r *Repo) InsertDecision(ctx context.Context, d domain.Decision) error {
	options, err := json.Marshal(d.Options)
	if err != nil {
		return err
	}
	if d.Options == nil {
		options = []byte("[]")
	}
	explanation, err := marshalExplanation(d.Explanation)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `INSERT INTO decisions (`+decisionColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		d.ID, d.ClientID, d.SourceTxID, d.Action, d.Amount, options, d.Reason, d.Status,
		d.CreatedAt, d.ExpiresAt, d.NotifyAt, explanation)
	return err
}

// UpdateDecision сохраняет изменения решения: статус, сумму, варианты, причину и время уведомления.
func (r *Repo) UpdateDecision(ctx context.Context, d domain.Decision, answeredAt *time.Time) error {
	options, err := json.Marshal(d.Options)
	if err != nil {
		return err
	}
	if d.Options == nil {
		options = []byte("[]")
	}
	explanation, err := marshalExplanation(d.Explanation)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE decisions SET action = $2, amount = $3, options = $4, reason = $5, status = $6,
		       expires_at = $7, notify_at = $8, answered_at = COALESCE($9, answered_at), explanation = $10
		WHERE id = $1`,
		d.ID, d.Action, d.Amount, options, d.Reason, d.Status, d.ExpiresAt, d.NotifyAt, answeredAt, explanation)
	return err
}

// GetDecisionForUpdate читает решение и блокирует его до конца транзакции:
// два одновременных ответа на один вопрос не исполнятся дважды.
func (r *Repo) GetDecisionForUpdate(ctx context.Context, id string) (domain.Decision, error) {
	return scanDecision(r.db.QueryRow(ctx, `SELECT `+decisionColumns+` FROM decisions WHERE id = $1 FOR UPDATE`, id))
}

func (r *Repo) GetDecisionBySourceTx(ctx context.Context, txID string) (domain.Decision, error) {
	return scanDecision(r.db.QueryRow(ctx, `SELECT `+decisionColumns+` FROM decisions WHERE source_tx_id = $1`, txID))
}

// ActiveQuestion — открытый вопрос клиенту (pending), если есть. Нет → domain.ErrNotFound.
func (r *Repo) ActiveQuestion(ctx context.Context, clientID string) (domain.Decision, error) {
	return scanDecision(r.db.QueryRow(ctx, `SELECT `+decisionColumns+` FROM decisions
		WHERE client_id = $1 AND status = 'pending' AND action IN ('ask', 'confirm_income')
		ORDER BY created_at LIMIT 1 FOR UPDATE`, clientID))
}

// NextQueued — самый ранний вопрос в очереди. Нет → domain.ErrNotFound.
func (r *Repo) NextQueued(ctx context.Context, clientID string) (domain.Decision, error) {
	return scanDecision(r.db.QueryRow(ctx, `SELECT `+decisionColumns+` FROM decisions
		WHERE client_id = $1 AND status = 'queued' ORDER BY created_at, id LIMIT 1 FOR UPDATE`, clientID))
}

// ListDecisions — решения клиента, новые первыми; status == "" — все.
func (r *Repo) ListDecisions(ctx context.Context, clientID string, status domain.DecisionStatus, limit int) ([]domain.Decision, error) {
	rows, err := r.db.Query(ctx, `SELECT `+decisionColumns+` FROM decisions
		WHERE client_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC, id DESC LIMIT $3`, clientID, string(status), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Decision{}
	for rows.Next() {
		d, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ExpiredQuestionIDs — открытые вопросы, на которые время ответа вышло к моменту now.
func (r *Repo) ExpiredQuestionIDs(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM decisions
		WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at <= $1
		ORDER BY expires_at, id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// OpenQuestionIDs — все открытые и ожидающие вопросы клиента (для закрытия при аресте).
func (r *Repo) OpenQuestionIDs(ctx context.Context, clientID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM decisions
		WHERE client_id = $1 AND status IN ('pending', 'queued') AND action IN ('ask', 'confirm_income')
		ORDER BY created_at, id`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
