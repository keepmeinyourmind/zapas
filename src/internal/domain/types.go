package domain

import (
	"errors"
	"fmt"
	"time"
)

// ------------- ДЕНЬГИ --------------------

type Money int64

func FromRubles(rubles int64) Money {
	return Money(rubles * 100)
}

func (m Money) Share(p int) Money {
	return m * Money(p) / 100
}

func (m Money) String() string {
	sign := ""
	if m < 0 {
		sign = "-"
		m = -m
	}
	rub, kop := int64(m)/100, int64(m)%100

	s := fmt.Sprintf("%d", rub)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + " " + s[i:]
	}
	if kop != 0 {
		return fmt.Sprintf("%s%s,%02d ₽", sign, s, kop)
	}
	return sign + s + " ₽"
}

// --------------- ТРАНЗАКЦИИ ----------------

// TxKind — тип операции.
type TxKind string

const (
	TxKindIncoming TxKind = "incoming"
	TxKindOutgoing TxKind = "outgoing"
	TxKindRefund   TxKind = "refund"
	TxKindInternal TxKind = "internal"
	TxKindLoan     TxKind = "loan"
)

// CounterpartyKind — вид отправителя или получателя денег.
type CounterpartyKind string

const (
	CounterpartyLegal  CounterpartyKind = "legal"
	CounterpartyPerson CounterpartyKind = "person"
	CounterpartySelf   CounterpartyKind = "self"
	CounterpartyBank   CounterpartyKind = "bank"
)

// Transaction — одна операция по карте.
type Transaction struct {
	ID               string           `json:"id"`
	ClientID         string           `json:"client_id"`
	Amount           Money            `json:"amount"`
	Type             TxKind           `json:"type"`
	Counterparty     string           `json:"counterparty"`
	CounterpartyKind CounterpartyKind `json:"counterparty_kind"`
	OccurredAt       time.Time        `json:"occurred_at"`
}

// IncomeKind — результат работы классификатора доходов.
type IncomeKind string

const (
	IncomeSalary      IncomeKind = "salary"
	IncomeAdvance     IncomeKind = "advance"
	IncomeBonus       IncomeKind = "bonus"
	IncomeScholarship IncomeKind = "scholarship"
	IncomeFreelance   IncomeKind = "freelance"
	IncomeNone        IncomeKind = "none"
)

func (k IncomeKind) IsIncome() bool {
	return k != IncomeNone && k != ""
}

// SavingsRule — правило накопления на цель.
type SavingsRule struct {
	ClientID       string `json:"client_id"`
	GoalName       string `json:"goal_name"`
	GoalAmount     Money  `json:"goal_amount"`
	TermMonths     int    `json:"term_months"`
	SavePercent    int    `json:"save_percent"`
	MinIncome      Money  `json:"min_income"`
	MinTransfer    Money  `json:"min_transfer"`
	DeclaredIncome Money  `json:"declared_income"`
	Paused         bool   `json:"paused"`
	QuietFrom      string `json:"quiet_from"` // тихие часы: с какого времени не беспокоить, "23:00"
	QuietTo        string `json:"quiet_to"`   // до какого времени, "09:00"
	Timezone       string `json:"timezone"`   // часовой пояс клиента, "Europe/Moscow"
}

const (
	DefaultMinTransfer Money = 100 * 100
	MaxPercent         int   = 50
)

func (r SavingsRule) Validate() error {
	switch {
	case r.ClientID == "":
		return errors.New("Не указан клиент")
	case r.GoalAmount <= 0:
		return errors.New("Сумма цели должна быть больше нуля")
	case r.TermMonths != 6 && r.TermMonths != 12 && r.TermMonths != 24:
		return errors.New("Срок должен быть 6, 12 или 24 месяца")
	case r.SavePercent < 1 || r.SavePercent > MaxPercent:
		return fmt.Errorf("Процент должен быть от 1 до %d", MaxPercent)
	case r.MinIncome < 0 || r.MinTransfer < 0 || r.DeclaredIncome < 0:
		return errors.New("Суммы не могут быть отрицательными")
	}
	return nil
}

type TrustLevel int

const (
	TrustLevelLow  TrustLevel = iota // < 3 месяцев истории: низкий уровень доверия, всегда спрашиваем
	TrustLevelMid                    // 3-5 месяцев: средний уровень доверия
	TrustLevelHigh                   // > 6 месяцев: высокий уровень доверия
)

// ------------ Решение по операции ----------------

type DecisionAction string

const (
	DecisionAuto          DecisionAction = "auto"
	DecisionAsk           DecisionAction = "ask"
	DecisionSkip          DecisionAction = "skip"
	DecisionSuggestRaise  DecisionAction = "suggest_raise"
	DecisionConfirmIncome DecisionAction = "confirm_income" // вопрос «это ваш доход?»
)

type DecisionStatus string

const (
	DecisionQueued   DecisionStatus = "queued" // ждёт, пока клиент ответит на предыдущий вопрос
	DecisionPending  DecisionStatus = "pending"
	DecisionExecuted DecisionStatus = "executed"
	DecisionSkipped  DecisionStatus = "skipped"
	DecisionExpired  DecisionStatus = "expired"
)

type Decision struct {
	ID          string         `json:"id"`
	ClientID    string         `json:"client_id"`
	SourceTxID  string         `json:"source_tx_id"`
	Action      DecisionAction `json:"action"`
	Amount      Money          `json:"amount"`
	Options     []Money        `json:"options,omitempty"`
	Reason      string         `json:"reason"`
	Status      DecisionStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	NotifyAt    *time.Time     `json:"notify_at,omitempty"`   // когда уведомить клиента с учётом тихих часов
	Explanation *Explanation   `json:"explanation,omitempty"` // из чего сложилась сумма; нет — если считать нечего
}

// ---------- Перевод ----------

type Transfer struct {
	ID         string    `json:"id"`
	DecisionID string    `json:"decision_id"`
	SourceTxID string    `json:"source_tx_id"`
	ClientID   string    `json:"client_id"`
	Amount     Money     `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
}
