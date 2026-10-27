package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/keepmeinyourmind/zapas/src/internal/classifier"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// TransactionResult — что вернуть по принятой транзакции.
type TransactionResult struct {
	IncomeKind domain.IncomeKind `json:"income_kind"`
	Decision   *domain.Decision  `json:"decision"` // null, если поступление не доход (решения подключим позже)
}

// Допустимые значения type и counterparty_kind — как в openapi.yaml.
var (
	knownTxKinds = map[domain.TxKind]bool{
		domain.TxKindIncoming: true, domain.TxKindOutgoing: true, domain.TxKindRefund: true,
		domain.TxKindInternal: true, domain.TxKindLoan: true,
	}
	knownCounterpartyKinds = map[domain.CounterpartyKind]bool{
		domain.CounterpartyLegal: true, domain.CounterpartyPerson: true,
		domain.CounterpartySelf: true, domain.CounterpartyBank: true,
	}
)

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
	case !knownTxKinds[tx.Type]:
		return domain.ValidationError(fmt.Sprintf("Неизвестный тип операции %q", tx.Type))
	case tx.CounterpartyKind == "":
		return domain.ValidationError("Не указан вид отправителя или получателя")
	case !knownCounterpartyKinds[tx.CounterpartyKind]:
		return domain.ValidationError(fmt.Sprintf("Неизвестный вид отправителя или получателя %q", tx.CounterpartyKind))
	case tx.OccurredAt.IsZero():
		return domain.ValidationError("Не указано время операции")
	}
	return nil
}

// ProcessTransaction — главный конвейер: сохранить → обновить баланс → классифицировать →
// решить → перевести. Всё в одной транзакции БД: либо всё, либо ничего.
// created=false — это повтор уже обработанной транзакции (идемпотентность по id): второго
// решения и второго перевода не будет, вернётся то, что решили в первый раз.
func (s *Service) ProcessTransaction(ctx context.Context, tx domain.Transaction) (res TransactionResult, created bool, err error) {
	if err := validateTransaction(tx); err != nil {
		return res, false, err
	}

	err = s.store.InTx(ctx, func(r *storage.Repo) error {
		// Блокируем клиента: его операции обрабатываются строго по очереди.
		client, err := r.GetClientForUpdate(ctx, tx.ClientID)
		if err != nil {
			return err
		}
		inserted, err := r.InsertTransaction(ctx, tx)
		if err != nil {
			return err
		}
		if !inserted { // повтор: отдаём то, что решили в первый раз
			// id глобальный: тот же id с другим клиентом или другими данными — не повтор,
			// а конфликт. Иначе отдали бы чужое решение.
			stored, err := r.GetTransaction(ctx, tx.ID)
			if err != nil {
				return err
			}
			if stored.ClientID != tx.ClientID || stored.Amount != tx.Amount || stored.Type != tx.Type {
				return domain.ErrIDConflict
			}
			if res.IncomeKind, err = r.GetIncomeKind(ctx, tx.ID); err != nil {
				return err
			}
			if d, err := r.GetDecisionBySourceTx(ctx, tx.ID); err == nil {
				res.Decision = &d
			} else if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			return nil
		}
		created = true

		delta := cardDelta(tx)
		if err := r.AddCardBalance(ctx, tx.ClientID, delta); err != nil {
			return err
		}
		client.CardBalance += delta

		// В простом режиме клиент не давал согласия на анализ — классификатор не запускаем.
		res.IncomeKind = domain.IncomeNone
		var confirmed map[string]bool
		if client.Mode != domain.ModeSimple {
			history, err := r.ListIncomeHistory(ctx, tx.ClientID, tx.OccurredAt.Add(-classifier.Window))
			if err != nil {
				return err
			}
			// Саму новую операцию из истории убираем
			history = slices.DeleteFunc(history, func(h domain.ClassifiedTransaction) bool { return h.ID == tx.ID })
			if confirmed, err = r.ConfirmedSenders(ctx, tx.ClientID); err != nil {
				return err
			}
			res.IncomeKind = classifier.Classify(tx, history, confirmed)
			if err := r.SetIncomeKind(ctx, tx.ID, res.IncomeKind); err != nil {
				return err
			}
		}

		res.Decision, err = s.decide(ctx, r, client, tx, res.IncomeKind, confirmed)
		return err
	})
	return res, created, err
}

// cardDelta — как операция меняет баланс карты.
func cardDelta(tx domain.Transaction) domain.Money {
	switch tx.Type {
	case domain.TxKindOutgoing:
		return -tx.Amount
	case domain.TxKindLoan:
		// Кредит — не свои деньги клиента: CardBalance — это свои деньги на карте,
		// а кредитные в расчёте безопасной суммы не участвуют.
		return 0
	}
	return tx.Amount
}

func (s *Service) ListTransactions(ctx context.Context, clientID string, limit int) ([]domain.ClassifiedTransaction, error) {
	repo := s.store.Read()
	if _, err := repo.GetClient(ctx, clientID); err != nil {
		return nil, err
	}
	return repo.ListClassified(ctx, clientID, limit)
}
