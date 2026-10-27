package profile

import (
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// incomeCycle — через сколько ждём следующую выплату того же вида.
const incomeCycle = 30 * day

// Options — что, кроме истории, нужно знать для профиля.
type Options struct {
	CurrentTxID    string          // поступление, по которому решаем: не входит в обычный доход и Previous
	DeclaredIncome domain.Money    // доход из анкеты: при низком доверии вместо медианы зарплаты
	Excluded       map[string]bool // платежи, которые клиент пометил необязательными: это обычные траты
}

// Build собирает профиль клиента по истории операций (уже классифицированных).
// Доход из анкеты при низком доверии используется вместо медианы зарплаты,
// потому что по одной-двум выплатам «обычную» сумму не понять.
// Возвращает все найденные регулярные платежи, и исключённые тоже: какие из них резервировать, решает сервис.

func Build(history []domain.ClassifiedTransaction, opts Options, now time.Time) (domain.Profile, []domain.RecurringPayment) {
	plain := make([]domain.Transaction, len(history))
	for i, tx := range history {
		plain[i] = tx.Transaction
	}

	months := MonthsOfData(history, now)
	p := domain.Profile{
		MonthsOfData: months,
		Trust:        Trust(months),
		Expected:     ExpectedIncome(history, opts.CurrentTxID, now),
		Previous:     PreviousIncome(history, opts.CurrentTxID, now),
		NextIncomeAt: NextIncomeAt(history, now),
	}
	if p.Trust == domain.TrustLevelLow && opts.DeclaredIncome > 0 {
		p.Expected[domain.IncomeSalary] = opts.DeclaredIncome
	}

	recurring := DetectRecurring(plain, now)
	// Исключённый платёж из повседневных трат не вычитаем: иначе его деньги нигде не учтены.
	var mandatory []domain.RecurringPayment
	for _, rp := range recurring {
		if !opts.Excluded[rp.Counterparty] {
			mandatory = append(mandatory, rp)
		}
	}
	p.DailySpending = DailySpending(plain, mandatory, now)
	return p, recurring
}

// regularIncome — виды дохода, которые приходят по графику. Премия и фриланс разовые:
// по ним следующую выплату не предскажешь, и горизонт расчёта они не сдвигают.
var regularIncome = map[domain.IncomeKind]bool{
	domain.IncomeSalary: true, domain.IncomeAdvance: true, domain.IncomeScholarship: true,
}

// NextIncomeAt — когда ждём следующий доход: для каждого регулярного вида дохода последняя выплата + 30 дней,
// берём ближайшую дату после now. У Ани аванс 10-го и зарплата 25-го: после зарплаты 25 октября
// следующий доход — аванс около 9 ноября, а не зарплата 24 ноября. Нет регулярного дохода — now + 30 дней.
func NextIncomeAt(history []domain.ClassifiedTransaction, now time.Time) time.Time {
	last := map[domain.IncomeKind]time.Time{}
	for _, tx := range history {
		if regularIncome[tx.IncomeKind] && !tx.OccurredAt.After(now) && tx.OccurredAt.After(last[tx.IncomeKind]) {
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
