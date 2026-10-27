package domain

import "time"

// Profile — что система знает о доходе клиента. Строит пакет profile, использует engine.
type Profile struct {
	Trust         TrustLevel
	MonthsOfData  int                  // сколько календарных месяцев с доходом в истории
	Expected      map[IncomeKind]Money // обычная сумма каждого вида дохода (медиана)
	NextIncomeAt  time.Time            // когда ждём следующий доход
	DailySpending Money                // средние повседневные траты в день, без регулярных платежей
}

// ExpectedFor — обычная сумма для вида дохода; 0 — если не знаем.
func (p Profile) ExpectedFor(k IncomeKind) Money { return p.Expected[k] }

// RecurringPayment — регулярный обязательный платёж: рассрочка, связь, подписка.
type RecurringPayment struct {
	Counterparty string
	Amount       Money     // обычная сумма платежа (медиана)
	DayOfMonth   int       // в какой день месяца обычно списывается
	LastAt       time.Time // когда списался последний раз
}

// Due — одно предстоящее обязательное списание.
type Due struct {
	Counterparty string
	Amount       Money
	At           time.Time
}

// Explanation — почему решение такое: из чего сложилась безопасная сумма.
// Фронтенд показывает список трат, summary — готовый текст для клиента.
type Explanation struct {
	CardBalance   Money          `json:"card_balance"`   // свои деньги на карте (с этим поступлением)
	UpcomingTotal Money          `json:"upcoming_total"` // сколько понадобится до следующего дохода
	Buffer        Money          `json:"buffer"`         // неприкосновенный остаток
	SafeAmount    Money          `json:"safe_amount"`    // сколько можно отложить безопасно
	Until         time.Time      `json:"until"`          // до какой даты считали (ожидаемый следующий доход)
	Upcoming      []UpcomingItem `json:"upcoming"`       // предстоящие траты по отдельности
	Summary       string         `json:"summary"`        // то же самое текстом
}

// UpcomingItem — одна предстоящая трата. У обязательного платежа есть дата,
// у повседневных трат — сумма в день и число дней.
type UpcomingItem struct {
	Title  string     `json:"title"`
	Amount Money      `json:"amount"`
	At     *time.Time `json:"at,omitempty"`
	PerDay Money      `json:"per_day,omitempty"`
	Days   int        `json:"days,omitempty"`
}
