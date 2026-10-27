// Package engine — движок решений: сколько отложить с поступления и спрашивать ли клиента.
// Decide — чистая функция: без БД, без time.Now(), одинаковый вход → одинаковое решение.
package engine

import (
	"fmt"
	"math"
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

	// Для расшифровки клиенту: из чего сложилась сумма Upcoming.
	Dues     []domain.Due // обязательные платежи до следующего дохода, по датам
	Spending domain.Money // повседневные траты до следующего дохода
	Days     int          // за сколько дней посчитаны повседневные траты
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
	// Безопасная сумма: только свои деньги на карте (кредитный лимит не считаем)
	// минус всё, что понадобится до следующего дохода, минус неприкосновенный буфер.
	safe := in.Client.CardBalance - in.Upcoming - cfg.Buffer
	if safe < 0 {
		safe = 0
	}
	want := in.Tx.Amount.Share(in.Rule.SavePercent) // сколько клиент хочет откладывать
	amount := min(safe, want)

	if amount < in.Rule.MinTransfer {
		return skip("Впереди обязательные платежи — в этот раз не откладываем")
	}

	if in.Profile.Trust == domain.TrustLevelLow {
		return ask(amount, []domain.Money{amount}, fmt.Sprintf("Мы еще изучаем ваш доход. Отложить %s?", amount))
	}
	// Насколько поступление отличается от обычного дохода такого вида, в целых процентах.
	// Аванс сравниваем с авансами, зарплату — с зарплатами. Не с чем сравнить — считаем обычным.
	expected := in.Profile.ExpectedFor(in.Kind)
	dev := 0
	if expected > 0 {
		dev = int((in.Tx.Amount - expected) * 100 / expected)
	}

	// 1. Премия — разовая выплата: предлагаем отложить больше обычного, если безопасно.
	if in.Kind == domain.IncomeBonus {
		options := []domain.Money{amount}
		if more := min(want*2, safe); more > amount {
			options = append(options, more)
		}
		return ask(amount, options, "Пришла премия — отложить больше в этот раз?")
	}

	// 2. Доход заметно ниже обычного — не списываем молча, спрашиваем. Главная фишка «Запаса».
	if dev < -cfg.DropPct {
		return ask(amount, []domain.Money{amount},
			fmt.Sprintf("Доход ниже обычного на %d%%. Отложить %s?", -dev, amount))
	}

	// 3. Впереди платёж: безопасно отложить меньше, чем обычно, — спрашиваем и объясняем почему.
	if amount < want {
		return ask(amount, []domain.Money{amount}, upcomingReason(in, amount))
	}

	// 4. Доход заметно выше обычного. Доверяем выводам — переводим и предлагаем поднять процент,
	// пока доверие среднее — спрашиваем.
	if dev > cfg.RaisePct {
		if in.Profile.Trust == domain.TrustLevelHigh {
			return suggestRaise(amount, fmt.Sprintf("Доход вырос на %d%% — откладываем %s. Можно поднять процент", dev, amount))
		}
		return ask(amount, []domain.Money{amount}, fmt.Sprintf("Доход выше обычного на %d%%. Отложить %s?", dev, amount))
	}
	return auto(amount, fmt.Sprintf("Доход в пределах обычного — откладываем %s", amount))
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

func ask(amount domain.Money, options []domain.Money, reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionAsk, Amount: amount, Options: options, Status: domain.DecisionPending, Reason: reason}
}

// suggestRaise — перевести и предложить клиенту поднять процент. Статус pending: перевод исполнит сервис.
func suggestRaise(amount domain.Money, reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionSuggestRaise, Amount: amount, Status: domain.DecisionPending, Reason: reason}
}

// upcomingReason — почему предлагаем отложить меньше обычного: называем ближайший платёж, если знаем его.
func upcomingReason(in Input, amount domain.Money) string {
	if in.Nearest == nil {
		return fmt.Sprintf("Впереди обязательные платежи — предлагаем отложить %s", amount)
	}
	days := int(math.Ceil(in.Nearest.At.Sub(in.Now).Hours() / 24))
	return fmt.Sprintf("Через %d дн. платёж «%s» — предлагаем отложить %s", days, in.Nearest.Counterparty, amount)
}
