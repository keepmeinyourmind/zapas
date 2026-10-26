package service

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// AccountState — состояние счёта после события банка.
type AccountState struct {
	ClientID string `json:"client_id"`
	Arrested bool   `json:"arrested"`
}

// ProcessAccountEvent применяет событие банка (арест / снятие ареста).
// created=false — повтор события с тем же id, ничего не меняется.
func (s *Service) ProcessAccountEvent(ctx context.Context, e domain.AccountEvent) (st AccountState, created bool, err error) {
	switch {
	case e.ID == "":
		return st, false, domain.ValidationError("Не указан id события")
	case e.ClientID == "":
		return st, false, domain.ValidationError("Не указан клиент")
	case e.OccurredAt.IsZero():
		return st, false, domain.ValidationError("Не указано время события")
	case e.Type != domain.EventArrestImposed && e.Type != domain.EventArrestLifted:
		return st, false, domain.ValidationError("Неизвестный тип события")
	}

	err = s.store.InTx(ctx, func(r *storage.Repo) error {
		if _, err := r.GetClient(ctx, e.ClientID); err != nil {
			return err
		}
		inserted, err := r.InsertAccountEvent(ctx, e)
		if err != nil {
			return err
		}
		if inserted {
			created = true
			// Открытые вопросы при аресте закроем, когда появятся решения (сервис переводов).
			if err := r.SetArrested(ctx, e.ClientID, e.Type == domain.EventArrestImposed, e.OccurredAt); err != nil {
				return err
			}
		}
		c, err := r.GetClient(ctx, e.ClientID)
		st = AccountState{ClientID: c.ID, Arrested: c.Arrested}
		return err
	})
	return st, created, err
}
