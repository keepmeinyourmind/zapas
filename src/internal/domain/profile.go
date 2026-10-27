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
