package service

import (
	"context"

	"github.com/keepmeinyourmind/zapas/src/internal/classifier"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// TransactionResult — что вернуть по принятой транзакции.
type TransactionResult struct {
	IncomeKind domain.IncomeKind `json:"income_kind"`
	Decision   *domain.Decision  `json:"decision"` // null, если поступление не доход (решения подключим позже)
}

func validateTransaction(tx domain.Transaction) error {
	switch {
	case tx.ID == "":
		return domain.ValidationError("Не указан id транзакции")
	case tx.ClientID == "":
		return domain.ValidationError("Не указан клиент")
	case tx.Amount <= 0:
		return domain.ValidationError("Сумма должна быть больше нуля")
	case tx.Type == "":
		return domain.ValidationError("Не указан тип операции")
	case tx.OccurredAt.IsZero():
		return domain.ValidationError("Не указано время операции")
	}
	return nil
}

// ProcessTransaction принимает транзакцию: сохранить → классифицировать.
// created=false — это повтор уже обработанной транзакции (идемпотентность по id).
func (s *Service) ProcessTransaction(ctx context.Context, tx domain.Transaction) (res TransactionResult, created bool, err error) {
	if err := validateTransaction(tx); err != nil {
		return res, false, err
	}

	err = s.store.InTx(ctx, func(r *storage.Repo) error {
		if _, err := r.GetClient(ctx, tx.ClientID); err != nil {
			return err
		}
		inserted, err := r.InsertTransaction(ctx, tx)
		if err != nil {
			return err
		}
		if !inserted { // повтор: отдаём то, что решили в первый раз
			res.IncomeKind, err = r.GetIncomeKind(ctx, tx.ID)
			return err
		}

		history, err := r.ListHistory(ctx, tx.ClientID, tx.ID)
		if err != nil {
			return err
		}
		confirmed, err := r.ConfirmedSenders(ctx, tx.ClientID)
		if err != nil {
			return err
		}
		res.IncomeKind = classifier.Classify(tx, history, confirmed)
		created = true
		return r.SetIncomeKind(ctx, tx.ID, res.IncomeKind)
	})
	return res, created, err
}

func (s *Service) ListTransactions(ctx context.Context, clientID string, limit int) ([]domain.ClassifiedTransaction, error) {
	repo := s.store.Read()
	if _, err := repo.GetClient(ctx, clientID); err != nil {
		return nil, err
	}
	return repo.ListClassified(ctx, clientID, limit)
}
