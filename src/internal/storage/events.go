package storage

import (
	"context"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// InsertAccountEvent сохраняет событие банка. inserted=false — повтор события с тем же id.
func (r *Repo) InsertAccountEvent(ctx context.Context, e domain.AccountEvent) (inserted bool, err error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO account_events (id, client_id, type, occurred_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`, e.ID, e.ClientID, e.Type, e.OccurredAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SetArrested меняет флаг ареста, только если событие новее последнего применённого:
// события могут прийти не по порядку, старое не должно отменить новое.
func (r *Repo) SetArrested(ctx context.Context, clientID string, arrested bool, at time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE clients SET arrested = $2, arrest_changed_at = $3
		WHERE id = $1 AND (arrest_changed_at IS NULL OR arrest_changed_at < $3)`, clientID, arrested, at)
	return err
}
