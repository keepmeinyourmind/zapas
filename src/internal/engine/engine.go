// Package engine — движок решений: сколько отложить с поступления и спрашивать ли клиента.
// Decide — чистая функция: без БД, без time.Now(), одинаковый вход → одинаковое решение.
package engine

import (
	"fmt"
	"math"
	"strings"
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

// skip — решение «пропустить» с причиной для клиента. Статус skipped: переводить нечего.
func skip(reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionSkip, Status: domain.DecisionSkipped, Reason: reason}
}

// explain — расшифровка для клиента: сколько на карте, какие траты впереди, сколько остаётся.
func explain(in Input, cfg Config, safe domain.Money) *domain.Explanation {
	e := &domain.Explanation{
		CardBalance:   in.Client.CardBalance,
		UpcomingTotal: in.Upcoming,
		Buffer:        cfg.Buffer,
		SafeAmount:    safe,
		Until:         in.Profile.NextIncomeAt,
		Upcoming:      []domain.UpcomingItem{}, // трат нет — фронтенд получит [], а не null
	}

	for _, due := range in.Dues {
		item := domain.UpcomingItem{Title: due.Counterparty, Amount: due.Amount}
		if !due.At.IsZero() {
			at := due.At
			item.At = &at
		}
		e.Upcoming = append(e.Upcoming, item)
	}

	if in.Spending > 0 {
		e.Upcoming = append(e.Upcoming, domain.UpcomingItem{
			Title: "Повседневные траты", Amount: in.Spending, PerDay: in.Profile.DailySpending, Days: in.Days,
		})
	}

	e.Summary = summary(e, in.Rule.MinTransfer, clientLocation(in.Rule.Timezone))
	return e
}

// clientLocation — часовой пояс клиента для дат в текстах. Не задан или неизвестен — UTC.
func clientLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// summary — расшифровка текстом, как её прочитает клиент. Даты — в часовом поясе клиента:
// платёж в 22:30 по UTC для москвича уже на следующий день.
func summary(e *domain.Explanation, minTransfer domain.Money, loc *time.Location) string {
	var lines []string

	if e.SafeAmount >= minTransfer {
		lines = append(lines, fmt.Sprintf("Можно безопасно отложить до %s.", e.SafeAmount))
	} else {
		lines = append(lines, "Сейчас безопасно отложить нечего.")
	}

	lines = append(lines, fmt.Sprintf("До %s предстоят траты на %s:", ruDate(e.Until.In(loc)), e.UpcomingTotal))
	for _, it := range e.Upcoming {
		switch {
		case it.At != nil:
			lines = append(lines, fmt.Sprintf("• %s — %s, %s", it.Title, it.Amount, ruDate(it.At.In(loc))))
		case it.Days > 0:
			lines = append(lines, fmt.Sprintf("• %s — %s (≈ %s в день × %d дн.)", it.Title, it.Amount, it.PerDay, it.Days))
		default:
			lines = append(lines, fmt.Sprintf("• %s — %s", it.Title, it.Amount))
		}
	}

	calc := fmt.Sprintf("На карте %s − %s", e.CardBalance, e.UpcomingTotal)
	if e.Buffer > 0 {
		calc += fmt.Sprintf(" − неприкосновенный остаток %s", e.Buffer)
	}
	// Денег меньше, чем нужно, — показываем настоящий минус, а не 0: так видно, сколько не хватает.
	if raw := e.CardBalance - e.UpcomingTotal - e.Buffer; raw < 0 {
		lines = append(lines, fmt.Sprintf("%s = %s, поэтому отложить нечего.", calc, raw))
	} else {
		lines = append(lines, fmt.Sprintf("%s = %s.", calc, e.SafeAmount))
	}

	return strings.Join(lines, "\n")
}

// ruMonths — месяцы в родительном падеже: Go не знает русских названий.
var ruMonths = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}

// ruDate — дата по-русски: «25 октября».
func ruDate(t time.Time) string {
	return fmt.Sprintf("%d %s", t.Day(), ruMonths[t.Month()-1])
}

// Decide принимает решение по поступлению.
// Заполняет Action, Amount, Options, Reason, Status. ID, время, notify_at и expires_at ставит сервис.
func Decide(in Input, cfg Config) domain.Decision {
	// 1. Арест: по закону переводы со счёта невозможны — это важнее всего остального.
	if in.Client.Arrested {
		return skip("Счёт арестован, переводы невозможны")
	}

	// 2. Клиент сам поставил копилку на паузу.
	if in.Rule.Paused {
		return skip("Копилка на паузе")
	}

	// 3. Цель уже накоплена: до неё осталось меньше минимального перевода — добавить нечего.
	if goalLeft(in) < in.Rule.MinTransfer {
		return skip("Цель достигнута — поздравляем!")
	}

	// Простой режим: клиент отказался от анализа — фиксированный процент, как автопополнение.
	// Стоит до проверки «не доход»: в простом режиме классификатор не запускается.
	if in.Client.Mode == domain.ModeSimple {
		return decideSimple(in)
	}

	// 4. Классификатор сказал, что это не доход.
	if !in.Kind.IsIncome() {
		return skip("Это поступление — не доход, его не откладываем")
	}

	// 5. Клиент просил не трогать маленькие поступления.
	if in.Tx.Amount < in.Rule.MinIncome {
		return skip(fmt.Sprintf("Поступления меньше %s не трогаем", in.Rule.MinIncome))
	}

	// Безопасная сумма: только свои деньги на карте (кредитный лимит не считаем)
	// минус всё, что понадобится до следующего дохода, минус неприкосновенный буфер.
	safe := in.Client.CardBalance - in.Upcoming - cfg.Buffer
	if safe < 0 {
		safe = 0
	}
	// Сколько клиент хочет откладывать, но не больше, чем осталось до цели.
	want := min(in.Tx.Amount.Share(in.Rule.SavePercent), goalLeft(in))
	amount := min(safe, want)

	d := decideAmount(in, cfg, safe, want, amount)
	d.Explanation = explain(in, cfg, safe)
	return d
}

// decideAmount — решение по сумме, когда безопасная сумма уже посчитана.
func decideAmount(in Input, cfg Config, safe, want, amount domain.Money) domain.Decision {
	if amount < in.Rule.MinTransfer {
		if safe < in.Rule.MinTransfer {
			return skip("Впереди обязательные платежи — в этот раз не откладываем")
		}
		// Деньги есть, просто доля дохода (или остаток до цели) маленькая.
		return skip(fmt.Sprintf("Откладывать меньше %s не будем", in.Rule.MinTransfer))
	}

	// Холодный старт: истории меньше 3 месяцев — своим выводам ещё не доверяем, спрашиваем.
	if in.Profile.Trust == domain.TrustLevelLow {
		return ask(amount, []domain.Money{amount}, fmt.Sprintf("Мы ещё изучаем ваш доход. Отложить %s?", amount))
	}
	// Насколько поступление отличается от обычного дохода такого вида.
	// Аванс сравниваем с авансами, зарплату — с зарплатами. Не с чем сравнить — считаем обычным.
	expected := in.Profile.ExpectedFor(in.Kind)
	drop := expected > 0 && below(in.Tx.Amount, expected, cfg.DropPct)
	rise := expected > 0 && above(in.Tx.Amount, expected, cfg.RaisePct)

	// 1. Премия — разовая выплата: предлагаем отложить больше обычного, если безопасно.
	if in.Kind == domain.IncomeBonus {
		options := []domain.Money{amount}
		if more := min(want*2, safe, goalLeft(in)); more > amount {
			options = append(options, more)
		}
		return ask(amount, options, "Пришла премия — отложить больше в этот раз?")
	}

	// 2. Доход заметно ниже обычного — не списываем молча, спрашиваем. Главная фишка «Запаса».
	if drop {
		return ask(amount, []domain.Money{amount},
			fmt.Sprintf("Доход ниже обычного на %d%%. Отложить %s?", -roundPct(in.Tx.Amount, expected), amount))
	}

	// 3. Впереди платёж: безопасно отложить меньше, чем обычно, — спрашиваем и объясняем почему.
	if amount < want {
		return ask(amount, []domain.Money{amount}, upcomingReason(in, amount))
	}

	// 4. Доход заметно выше обычного. Поднять процент предлагаем, только если доход вырос третий месяц
	// подряд и выводам доверяем: один-два высоких месяца могут быть случайностью — тогда спрашиваем.
	if rise {
		dev := roundPct(in.Tx.Amount, expected)
		if in.Profile.Trust == domain.TrustLevelHigh && risingBefore(in, expected, cfg.RaisePct) {
			return suggestRaise(amount, fmt.Sprintf("Доход вырос на %d%% — откладываем %s. Можно поднять процент", dev, amount))
		}
		return ask(amount, []domain.Money{amount}, fmt.Sprintf("Доход выше обычного на %d%%. Отложить %s?", dev, amount))
	}
	return auto(amount, fmt.Sprintf("Доход в пределах обычного — откладываем %s", amount))
}

// risingBefore — две прошлые выплаты этого вида тоже были выше обычного больше чем на pct процентов.
// Вместе с текущей это третий месяц роста подряд.
func risingBefore(in Input, expected domain.Money, pct int) bool {
	prev := in.Profile.Previous[in.Kind]
	if len(prev) < 2 {
		return false
	}
	for _, p := range prev[:2] {
		if !above(p, expected, pct) {
			return false
		}
	}
	return true
}

// below — amount ниже expected больше чем на pct процентов. Сравниваем точно, в целых числах:
// (amount − expected) × 100 < −expected × pct. Целые проценты обрезали бы 15,9% до 15
// и падение на 15,9% прошло бы как «в пределах обычного».
func below(amount, expected domain.Money, pct int) bool {
	return (amount-expected)*100 < -expected*domain.Money(pct)
}

// above — amount выше expected больше чем на pct процентов, тоже точно.
func above(amount, expected domain.Money, pct int) bool {
	return (amount-expected)*100 > expected*domain.Money(pct)
}

// roundPct — на сколько процентов amount отличается от expected, с округлением до целого: для текста.
func roundPct(amount, expected domain.Money) int {
	diff := (amount - expected) * 100
	half := expected / 2
	if diff < 0 {
		half = -half
	}
	return int((diff + half) / expected)
}

// goalLeft — сколько осталось до цели: больше откладывать незачем.
func goalLeft(in Input) domain.Money {
	return in.Rule.GoalAmount - in.Client.SavingsBalance
}

// auto — решение перевести сумму без вопроса. Статус pending: перевод ещё исполнит сервис.
func auto(amount domain.Money, reason string) domain.Decision {
	return domain.Decision{Action: domain.DecisionAuto, Amount: amount, Status: domain.DecisionPending, Reason: reason}
}

// decideSimple — простой режим: фиксированный процент с каждого входящего поступления.
func decideSimple(in Input) domain.Decision {
	if in.Tx.CounterpartyKind == domain.CounterpartySelf {
		return skip("Это перевод между вашими счетами — его не откладываем")
	}
	if in.Tx.CounterpartyKind == domain.CounterpartyBank {
		return skip("Деньги от банка — кешбэк и проценты — не откладываем")
	}
	if in.Tx.Amount < in.Rule.MinIncome {
		return skip(fmt.Sprintf("Поступления меньше %s не трогаем", in.Rule.MinIncome))
	}
	// Не больше, чем есть на карте, и не больше, чем осталось до цели.
	amount := min(in.Tx.Amount.Share(in.Rule.SavePercent), in.Client.CardBalance, goalLeft(in))
	if amount < in.Rule.MinTransfer {
		return skip(fmt.Sprintf("Откладывать меньше %s не будем", in.Rule.MinTransfer))
	}
	return auto(amount, fmt.Sprintf("Откладываем %d%% с поступления", in.Rule.SavePercent))
}

// ask — решение спросить клиента. options — варианты суммы на выбор; «пропустить» есть всегда.
// Статус pending: ждём ответа (срок ответа ставит сервис).
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
	if days <= 0 {
		return fmt.Sprintf("Сегодня платёж «%s» — предлагаем отложить %s", in.Nearest.Counterparty, amount)
	}
	return fmt.Sprintf("Через %d дн. платёж «%s» — предлагаем отложить %s", days, in.Nearest.Counterparty, amount)
}
