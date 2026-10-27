package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Варианты ответа клиента.
const (
	ChoiceAmount = "amount" // вопрос о сумме: перевести указанную сумму
	ChoiceSkip   = "skip"   // вопрос о сумме: пропустить
	ChoiceYes    = "yes"    // «это ваш доход?» — да
	ChoiceNo     = "no"     // «это ваш доход?» — нет
)

// Answer — ответ клиента на вопрос.
type Answer struct {
	Choice string       `json:"choice"`
	Amount domain.Money `json:"amount"`
}

// AnswerDecision принимает ответ клиента. Ответить можно только на активный вопрос;
// если время вышло — вопрос закрывается безопасным вариантом, ответ отклоняется.
func (s *Service) AnswerDecision(ctx context.Context, id string, a Answer) (domain.Decision, error) {
	var d domain.Decision
	err := s.store.InTx(ctx, func(r *storage.Repo) error {
		var err error
		if d, err = r.GetDecisionForUpdate(ctx, id); err != nil {
			return err
		}
		client, err := r.GetClientForUpdate(ctx, d.ClientID)
		if err != nil {
			return err
		}
		rule, err := r.GetRule(ctx, d.ClientID)
		if err != nil {
			return err
		}
		now := s.now()

		switch {
		case d.Status == domain.DecisionQueued:
			return fmt.Errorf("%w: сначала ответьте на предыдущий вопрос", domain.ErrDecisionClosed)
		case d.Status == domain.DecisionExpired:
			return fmt.Errorf("%w: %s", domain.ErrDecisionClosed, lowerFirst(d.Reason))
		case d.Status != domain.DecisionPending || !isQuestion(d):
			return fmt.Errorf("%w: ответ уже не нужен", domain.ErrDecisionClosed)
		case d.ExpiresAt != nil && !now.Before(*d.ExpiresAt):
			// Ответ опоздал: закрываем безопасным вариантом и сообщаем, что сделано.
			if err := s.expire(ctx, r, &d, rule, now); err != nil {
				return err
			}
			return fmt.Errorf("%w: время на ответ вышло — %s", domain.ErrDecisionClosed, lowerFirst(d.Reason))
		case client.Arrested:
			return domain.ErrAccountArrested
		}

		switch d.Action {
		case domain.DecisionAsk:
			err = s.answerAmount(ctx, r, &d, rule, a)
		case domain.DecisionConfirmIncome:
			err = s.answerConfirm(ctx, r, &d, client, rule, a, now)
		}
		if err != nil {
			return err
		}
		answered := now
		if err := r.UpdateDecision(ctx, d, &answered); err != nil {
			return err
		}
		return s.activateNext(ctx, r, d.ClientID, rule, now)
	})
	return d, err
}

func (s *Service) answerAmount(ctx context.Context, r *storage.Repo, d *domain.Decision, rule domain.SavingsRule, a Answer) error {
	switch a.Choice {
	case ChoiceSkip:
		d.Status = domain.DecisionSkipped
		d.Reason = "Вы решили в этот раз не откладывать"
		return nil
	case ChoiceAmount:
		limit := slices.Max(append(slices.Clone(d.Options), d.Amount)) // больше предложенного не переводим
		if a.Amount < rule.MinTransfer || a.Amount > limit {
			return domain.ValidationError(fmt.Sprintf("Сумма должна быть от %s до %s", rule.MinTransfer, limit))
		}
		d.Amount = a.Amount
		d.Reason = fmt.Sprintf("Вы решили отложить %s", a.Amount)
		return s.execute(ctx, r, d, rule)
	default:
		return domain.ValidationError("Для этого вопроса ответ — amount или skip")
	}
}

// answerConfirm — ответ на «это ваш доход?». «Да» — отправитель становится подтверждённым,
// поступление — фрилансом, и по нему сразу принимается решение о сумме (в том же вопросе).
func (s *Service) answerConfirm(ctx context.Context, r *storage.Repo, d *domain.Decision, client domain.Client,
	rule domain.SavingsRule, a Answer, now time.Time) error {

	switch a.Choice {
	case ChoiceNo:
		d.Status = domain.DecisionSkipped
		d.Reason = "Вы указали, что это не доход — не откладываем"
		return nil
	case ChoiceYes:
	default:
		return domain.ValidationError("Для этого вопроса ответ — yes или no")
	}

	tx, err := r.GetTransaction(ctx, d.SourceTxID)
	if err != nil {
		return err
	}
	if err := r.AddConfirmedSender(ctx, d.ClientID, tx.Counterparty); err != nil {
		return err
	}
	if err := r.SetIncomeKind(ctx, tx.ID, domain.IncomeFreelance); err != nil {
		return err
	}

	next, err := s.decideIncome(ctx, r, client, tx, domain.IncomeFreelance, rule, now)
	if err != nil {
		return err
	}
	// Тот же вопрос превращается в решение о сумме: у одного поступления одно решение.
	d.Action, d.Amount, d.Options, d.Reason = next.Action, next.Amount, next.Options, next.Reason
	d.Explanation = next.Explanation
	d.Status = next.Status
	switch {
	case d.Action == domain.DecisionAsk:
		s.setNotify(d, rule, now, true) // новый вопрос о сумме — снова 24 часа
	case d.Action == domain.DecisionAuto || d.Action == domain.DecisionSuggestRaise:
		return s.execute(ctx, r, d, rule)
	}
	return nil
}

// ExpireDue закрывает вопросы, на которые клиент не ответил вовремя, безопасным вариантом
// и уведомляет о следующем вопросе в очереди. Вызывается фоновым таймером и из демо.
func (s *Service) ExpireDue(ctx context.Context) ([]domain.Decision, error) {
	now := s.now()
	ids, err := s.store.Read().ExpiredQuestionIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	expired := []domain.Decision{}
	for _, id := range ids {
		err := s.store.InTx(ctx, func(r *storage.Repo) error {
			d, err := r.GetDecisionForUpdate(ctx, id)
			if err != nil {
				return err
			}
			if d.Status != domain.DecisionPending { // пока ждали блокировку, клиент успел ответить
				return nil
			}
			rule, err := r.GetRule(ctx, d.ClientID)
			if err != nil {
				return err
			}
			if err := s.expire(ctx, r, &d, rule, now); err != nil {
				return err
			}
			expired = append(expired, d)
			return s.activateNext(ctx, r, d.ClientID, rule, now)
		})
		if err != nil {
			return expired, err
		}
	}
	return expired, nil
}

// expire — безопасный вариант для вопроса без ответа: в этот раз ничего не откладываем.
func (s *Service) expire(ctx context.Context, r *storage.Repo, d *domain.Decision, rule domain.SavingsRule, now time.Time) error {
	d.Status = domain.DecisionExpired
	if d.Action == domain.DecisionConfirmIncome {
		d.Reason = "Вы не ответили — считаем, что это не доход, и не откладываем"
	} else {
		d.Reason = "Вы не ответили за 24 часа — в этот раз не откладываем"
	}
	return r.UpdateDecision(ctx, *d, nil)
}

// activateNext — если открытого вопроса нет, делает активным следующий из очереди.
func (s *Service) activateNext(ctx context.Context, r *storage.Repo, clientID string, rule domain.SavingsRule, now time.Time) error {
	if _, err := r.ActiveQuestion(ctx, clientID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	next, err := r.NextQueued(ctx, clientID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	s.setNotify(&next, rule, now, true)
	return r.UpdateDecision(ctx, next, nil)
}

// lowerFirst — строка с маленькой буквы, чтобы вставить причину в середину фразы.
func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}
