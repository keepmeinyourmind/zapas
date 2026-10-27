package profile

import (
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// incomeWindow — за какой период смотрим историю дохода: последние 6 месяцев.
const incomeWindow = 180 * day

// MonthsOfData — сколько разных календарных месяцев с доходом за последние 6 месяцев.
// Учитываются только поступления, которые классификатор признал доходом.
func MonthsOfData(history []domain.ClassifiedTransaction, now time.Time) int {
	from := now.Add(-incomeWindow)
	months := map[string]bool{}
	for _, tx := range history {
		if !tx.IncomeKind.IsIncome() || tx.OccurredAt.Before(from) {
			continue
		}
		key := tx.OccurredAt.Format("2006-01") // "2026-10": год и месяц
		months[key] = true
	}
	return len(months)
}

// ExpectedIncome — обычная сумма каждого вида дохода: медиана выплат этого вида за 6 месяцев.
// Аванс и зарплата считаются отдельно: аванс сравниваем с авансами, зарплату с зарплатами.
func ExpectedIncome(history []domain.ClassifiedTransaction, now time.Time) map[domain.IncomeKind]domain.Money {
	from := now.Add(-incomeWindow)
	byKind := map[domain.IncomeKind][]domain.Money{}
	for _, tx := range history {
		if !tx.IncomeKind.IsIncome() || tx.OccurredAt.Before(from) {
			continue
		}
		byKind[tx.IncomeKind] = append(byKind[tx.IncomeKind], tx.Amount)
	}

	expected := map[domain.IncomeKind]domain.Money{}
	for kind, amounts := range byKind {
		expected[kind] = domain.Median(amounts)
	}
	return expected
}
