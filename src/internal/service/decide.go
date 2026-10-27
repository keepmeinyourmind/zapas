package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/engine"
	"github.com/keepmeinyourmind/zapas/src/internal/notify"
	"github.com/keepmeinyourmind/zapas/src/internal/profile"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// profileWindow — сколько истории загружаем для профиля: 6 месяцев с запасом.
const profileWindow = 200 * 24 * time.Hour

// decide принимает решение по сохранённой операции. nil — решения нет: операция не входящая,
// копилка не подключена или (в умном режиме) это не доход.
func (s *Service) decide(ctx context.Context, r *storage.Repo, client domain.Client, tx domain.Transaction,
	kind domain.IncomeKind, confirmed map[string]bool) (*domain.Decision, error) {

	if tx.Type != domain.TxKindIncoming {
		return nil, nil
	}
	rule, err := r.GetRule(ctx, tx.ClientID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil // копилка не подключена
	}
	if err != nil {
		return nil, err
	}
	now := s.now()

	var d domain.Decision
	switch {
	case client.Mode == domain.ModeSimple || kind.IsIncome():
		if d, err = s.decideIncome(ctx, r, client, tx, kind, rule, now); err != nil {
			return nil, err
		}
		if d.Action == "" { // ветка движка ещё не написана — решения нет
			s.log.Warn("движок не вернул решение", "tx", tx.ID)
			return nil, nil
		}

	case !client.Arrested && tx.CounterpartyKind == domain.CounterpartyPerson && !confirmed[tx.Counterparty] &&
		!rule.Paused && client.SavingsBalance < rule.GoalAmount && tx.Amount >= rule.MinIncome:
		// Перевод от незнакомого физлица: не решаем молча, а спрашиваем «это ваш доход?».
		// Если откладывать всё равно не будем (пауза, цель достигнута, мелкое поступление) — не спрашиваем.
		d = domain.Decision{
			Action: domain.DecisionConfirmIncome, Amount: tx.Amount, Status: domain.DecisionPending,
			Reason: fmt.Sprintf("Перевод %s от «%s». Это ваш доход?", tx.Amount, tx.Counterparty),
		}

	default:
		return nil, nil // не доход — решения нет
	}

	d.ID = uuid.NewString()
	d.ClientID = tx.ClientID
	d.SourceTxID = tx.ID
	d.CreatedAt = now

	if err := s.schedule(ctx, r, &d, rule, now); err != nil {
		return nil, err
	}
	if err := r.InsertDecision(ctx, d); err != nil {
		return nil, err
	}
	if d.Action == domain.DecisionAuto || d.Action == domain.DecisionSuggestRaise {
		if err := s.execute(ctx, r, &d, rule); err != nil {
			return nil, err
		}
		if err := r.UpdateDecision(ctx, d, nil); err != nil {
			return nil, err
		}
	}
	return &d, nil
}

// decideIncome — профиль клиента → предстоящие расходы → движок.
func (s *Service) decideIncome(ctx context.Context, r *storage.Repo, client domain.Client, tx domain.Transaction,
	kind domain.IncomeKind, rule domain.SavingsRule, now time.Time) (domain.Decision, error) {

	history, err := r.ListIncomeHistory(ctx, tx.ClientID, now.Add(-profileWindow))
	if err != nil {
		return domain.Decision{}, err
	}
	declaredPayments, err := r.DeclaredPayments(ctx, tx.ClientID)
	if err != nil {
		return domain.Decision{}, err
	}
	excluded, err := r.ExcludedPayments(ctx, tx.ClientID)
	if err != nil {
		return domain.Decision{}, err
	}
	// Резервируем деньги только под обязательные платежи. Платёж, который клиент пометил
	// необязательным, — обычная трата: он входит в повседневные траты, а не в резерв по дате.
	prof, recurring := profile.Build(history, profile.Options{
		CurrentTxID: tx.ID, DeclaredIncome: rule.DeclaredIncome, Excluded: excluded,
	}, now)
	// Платежи из анкеты — только для холодного старта, пока по истории не найдено ни одного платежа.
	// Если клиент исключил все найденные, анкета их не заменяет.
	if len(recurring) > 0 {
		declaredPayments = 0
	}
	up := profile.Upcoming(withoutExcluded(recurring, excluded), declaredPayments, prof.DailySpending, now, prof.NextIncomeAt)

	return engine.Decide(engine.Input{
		Now: now, Tx: tx, Kind: kind, Rule: rule, Client: client,
		Profile: prof, Upcoming: up.Total, Nearest: up.Nearest,
		Dues: up.Dues, Spending: up.Spending, Days: up.Days,
	}, s.cfg), nil
}

func isQuestion(d domain.Decision) bool {
	return d.Action == domain.DecisionAsk || d.Action == domain.DecisionConfirmIncome
}

// schedule ставит время уведомления с учётом тихих часов, а вопросу — срок ответа и место в очереди.
// Активный вопрос у клиента один: новый вопрос о сумме объединяется с открытым вопросом о сумме,
// остальные встают в очередь и уведомляют, когда клиент ответит на предыдущий.
func (s *Service) schedule(ctx context.Context, r *storage.Repo, d *domain.Decision, rule domain.SavingsRule, now time.Time) error {
	if !isQuestion(*d) {
		s.setNotify(d, rule, now, false)
		return nil
	}

	active, err := r.ActiveQuestion(ctx, d.ClientID)
	if err == nil && active.ExpiresAt != nil && !now.Before(*active.ExpiresAt) {
		// Срок ответа вышел, а таймер ещё не успел закрыть вопрос: закрываем сами,
		// а не объединяем с новым, и смотрим, какой вопрос теперь активный.
		if err := s.expire(ctx, r, &active, rule, now); err != nil {
			return err
		}
		if err := s.activateNext(ctx, r, d.ClientID, rule, now); err != nil {
			return err
		}
		active, err = r.ActiveQuestion(ctx, d.ClientID)
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		s.setNotify(d, rule, now, true)
		return nil
	case err != nil:
		return err
	}

	if active.Action == domain.DecisionAsk && d.Action == domain.DecisionAsk {
		// Объединяем: один вопрос с общей суммой вместо двух уведомлений подряд.
		d.Amount += active.Amount
		d.Options = mergeOptions(active.Options, d.Options)
		if d.Explanation != nil {
			// Деньги прошлого поступления ещё на карте и уже учтены в новой безопасной сумме:
			// больше неё не предлагаем, иначе прошлое поступление посчитаем дважды.
			d.Amount = min(d.Amount, d.Explanation.SafeAmount)
			d.Options = capOptions(d.Options, d.Explanation.SafeAmount)
		}
		d.Reason = fmt.Sprintf("%s Вместе с прошлым поступлением предлагаем отложить %s.", d.Reason, d.Amount)
		active.Status = domain.DecisionExpired
		active.Reason = "Объединён с вопросом по новому поступлению"
		if err := r.UpdateDecision(ctx, active, nil); err != nil {
			return err
		}
		s.setNotify(d, rule, now, true)
		return nil
	}

	d.Status = domain.DecisionQueued // уведомим, когда клиент ответит на текущий вопрос
	d.NotifyAt, d.ExpiresAt = nil, nil
	return nil
}

// mergeOptions складывает варианты двух вопросов попарно: рекомендуемый с рекомендуемым и т. д.
func mergeOptions(a, b []domain.Money) []domain.Money {
	n := max(len(a), len(b))
	out := make([]domain.Money, n)
	for i := range n {
		out[i] = at(a, i) + at(b, i)
	}
	return out
}

// capOptions ограничивает варианты суммой limit и убирает повторы, которые от этого появились.
func capOptions(options []domain.Money, limit domain.Money) []domain.Money {
	out := make([]domain.Money, 0, len(options))
	for _, o := range options {
		if o = min(o, limit); !slices.Contains(out, o) {
			out = append(out, o)
		}
	}
	return out
}

// at — i-й элемент или последний, если слайс короче; 0 для пустого.
func at(xs []domain.Money, i int) domain.Money {
	if len(xs) == 0 {
		return 0
	}
	return xs[min(i, len(xs)-1)]
}

// setNotify — когда уведомить клиента: сразу или после тихих часов. У вопроса срок ответа
// считается от уведомления: ночью клиент спит, его 24 часа начинаются утром.
func (s *Service) setNotify(d *domain.Decision, rule domain.SavingsRule, now time.Time, question bool) {
	at := now
	if q, err := notify.ParseQuiet(rule.QuietFrom, rule.QuietTo, rule.Timezone); err == nil {
		at = notify.NotifyAt(now, q)
	}
	d.NotifyAt = &at
	if question {
		exp := at.Add(s.askTTL)
		d.ExpiresAt = &exp
		d.Status = domain.DecisionPending
	}
}

// execute исполняет перевод с карты в копилку. Баланс перепроверяется: между решением и переводом
// он мог измениться. Не хватает денег — переводим сколько есть; меньше минимума — не переводим.
func (s *Service) execute(ctx context.Context, r *storage.Repo, d *domain.Decision, rule domain.SavingsRule) error {
	client, err := r.GetClientForUpdate(ctx, d.ClientID)
	if err != nil {
		return err
	}
	if client.Arrested {
		d.Status = domain.DecisionSkipped
		d.Reason = "Счёт арестован, переводы невозможны"
		return nil
	}

	amount := min(d.Amount, client.CardBalance)
	if amount < rule.MinTransfer {
		d.Status = domain.DecisionSkipped
		d.Reason = fmt.Sprintf("На карте недостаточно денег: откладывать меньше %s не будем", rule.MinTransfer)
		return nil
	}

	inserted, err := r.InsertTransfer(ctx, domain.Transfer{
		ID: uuid.NewString(), DecisionID: d.ID, SourceTxID: d.SourceTxID,
		ClientID: d.ClientID, Amount: amount, CreatedAt: s.now(),
	})
	if err != nil {
		return err
	}
	if inserted { // перевод по этому поступлению уже был — деньги второй раз не двигаем
		if err := r.MoveToSavings(ctx, d.ClientID, amount); err != nil {
			return err
		}
	}
	d.Amount = amount
	d.Status = domain.DecisionExecuted
	return nil
}
