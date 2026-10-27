package profile

import (
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// incomeCycle — через сколько ждём следующую выплату того же вида.
const incomeCycle = 30 * day

// Build собирает профиль клиента по истории операций (уже классифицированных).
// declaredIncome — доход из анкеты: при низком доверии используется вместо медианы зарплаты,
// потому что по одной-двум выплатам «обычную» сумму не понять.
// Возвращает и найденные регулярные платежи — они нужны для Upcoming.
func Build(history []domain.ClassifiedTransaction, declaredIncome domain.Money, now time.Time) (domain.Profile, []domain.RecurringPayment) {
	plain := make([]domain.Transaction, len(history))
	for i, tx := range history {
		plain[i] = tx.Transaction
	}

	months := MonthsOfData(history, now)
	p := domain.Profile{
		MonthsOfData: months,
		Trust:        Trust(months),
		Expected:     ExpectedIncome(history, now),
		NextIncomeAt: NextIncomeAt(history, now),
	}
	if p.Trust == domain.TrustLevelLow && declaredIncome > 0 {
		p.Expected[domain.IncomeSalary] = declaredIncome
	}

	recurring := DetectRecurring(plain, now)
	p.DailySpending = DailySpending(plain, recurring, now)
	return p, recurring
}

// NextIncomeAt — когда ждём следующий доход: для каждого вида дохода последняя выплата + 30 дней,
// берём ближайшую дату после now. У Ани аванс 10-го и зарплата 25-го: после зарплаты 25 октября
// следующий доход — аванс около 9 ноября, а не зарплата 24 ноября. Нет истории — now + 30 дней.
func NextIncomeAt(history []domain.ClassifiedTransaction, now time.Time) time.Time {
	last := map[domain.IncomeKind]time.Time{}
	for _, tx := range history {
		if tx.IncomeKind.IsIncome() && !tx.OccurredAt.After(now) && tx.OccurredAt.After(last[tx.IncomeKind]) {
			last[tx.IncomeKind] = tx.OccurredAt
		}
	}

	var next time.Time
	for _, at := range last {
		due := at.Add(incomeCycle)
		if due.After(now) && (next.IsZero() || due.Before(next)) {
			next = due
		}
	}
	if next.IsZero() {
		return now.Add(incomeCycle)
	}
	return next
}
