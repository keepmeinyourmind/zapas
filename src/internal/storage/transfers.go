package storage

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// InsertTransfer записывает перевод в копилку. inserted=false — перевод по этому поступлению
// уже был (уникальный source_tx_id): повторное событие не даёт второго перевода.
func (r *Repo) InsertTransfer(ctx context.Context, t domain.Transfer) (inserted bool, err error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO transfers (id, decision_id, source_tx_id, client_id, amount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source_tx_id) DO NOTHING`,
		t.ID, t.DecisionID, t.SourceTxID, t.ClientID, t.Amount, t.CreatedAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
