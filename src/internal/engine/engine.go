// Package engine — движок решений: сколько отложить с поступления и спрашивать ли клиента.
// Decide — чистая функция: без БД, без time.Now(), одинаковый вход → одинаковое решение.
package engine

import (
	"fmt"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Config — пороги движка. Задаются в конфиге (переменные окружения), а не в коде.
type Config struct {
	DropPct  int          // доход ниже обычного больше чем на столько процентов → спросить клиента
	RaisePct int          // выше обычного больше чем на столько процентов → предложить поднять процент
	Buffer   domain.Money // неприкосновенный остаток на карте
}

// Input — всё, что движку нужно знать для решения. Собирает сервис.
type Input struct {
	Now      time.Time
	Tx       domain.Transaction // поступление, по которому решаем
	Kind     domain.IncomeKind  // что о нём сказал классификатор
	Rule     domain.SavingsRule
	Client   domain.Client  // CardBalance уже включает это поступление; CreditLimit не используем
	Profile  domain.Profile // обычный доход, уровень доверия
	Upcoming domain.Money   // сколько понадобится до следующего дохода (profile.Upcoming)
	Nearest  *domain.Due    // ближайший обязательный платёж, nil — если нет
}

func skip(reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionSkip, Status: domain.DecisionSkipped, Reason: reason}
}

// Decide принимает решение по поступлению.
// Заполняет Action, Amount, Options, Reason, Status. ID, время, notify_at и expires_at ставит сервис.
func Decide(in Input, cfg Config) domain.Decision {
	// 1. Арест: по закону переводы со счёта невозможны — это важнее всего остального.
	// 2. Клиент сам поставил копилку на паузу.
	// 3. Цель уже накоплена.
	// 4. Классификатор сказал, что это не доход.
	// 5. Клиент просил не трогать маленькие поступления.
	if in.Client.Arrested {
		return skip("Счёт арестован, переводы невозможны")
	}
	if in.Rule.Paused {
		return skip("Копилка на паузе")
	}
	if in.Client.SavingsBalance >= in.Rule.GoalAmount {
		return skip("Цель достигнута - поздравляем!")
	}
	if in.Client.Mode == domain.ModeSimple {
		return decideSimple(in)
	}
	if !in.Kind.IsIncome() {
		return skip("Это поступление - не доход, его не откладываем")
	}
	if in.Tx.Amount < in.Rule.MinIncome {
		return skip(fmt.Sprintf("Поступления меньше %s не трогаем", in.Rule.MinIncome))
	}
	return domain.Decision{}
}

// auto — решение перевести сумму без вопроса. Статус pending: перевод ещё исполнит сервис.
func auto(amount domain.Money, reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionAuto, Amount: amount, Status: domain.DecisionPending, Reason: reason}
}

// decideSimple — простой режим: фиксированный процент с каждого входящего поступления.
func decideSimple(in Input) domain.Decision {
	if in.Tx.Type != domain.TxKindIncoming || in.Tx.CounterpartyKind == domain.CounterpartySelf {
		return skip("Это перевод между вашими счетами — его не откладываем")
	}
	if in.Tx.Amount < in.Rule.MinIncome {
		return skip(fmt.Sprintf("Поступления меньше %s не трогаем", in.Rule.MinIncome))
	}
	amount := min(in.Tx.Amount.Share(in.Rule.SavePercent), in.Client.CardBalance) // не больше, чем есть на карте
	if amount < in.Rule.MinTransfer {
		return skip(fmt.Sprintf("Откладывать меньше %s не будем", in.Rule.MinTransfer))
	}
	return auto(amount, fmt.Sprintf("Откладываем %d%% с поступления", in.Rule.SavePercent))

}
