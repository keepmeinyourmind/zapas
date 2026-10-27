package profile

import (
	"slices"
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
// exceptID — поступление, по которому решаем: в медиану не входит, иначе оно сравнивается само с собой.
func ExpectedIncome(history []domain.ClassifiedTransaction, exceptID string, now time.Time) map[domain.IncomeKind]domain.Money {
	from := now.Add(-incomeWindow)
	byKind := map[domain.IncomeKind][]domain.Money{}
	for _, tx := range history {
		if !tx.IncomeKind.IsIncome() || tx.OccurredAt.Before(from) || isCurrent(tx, exceptID) {
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

// previousCount — сколько прошлых выплат каждого вида отдаём движку: ему нужны два прошлых месяца.
const previousCount = 2

// PreviousIncome — до двух последних выплат каждого вида дохода до now, от новой к старой,
// без поступления exceptID (по нему как раз решаем).
func PreviousIncome(history []domain.ClassifiedTransaction, exceptID string, now time.Time) map[domain.IncomeKind][]domain.Money {
	from := now.Add(-incomeWindow)
	var txs []domain.ClassifiedTransaction
	for _, tx := range history {
		if tx.IncomeKind.IsIncome() && !tx.OccurredAt.Before(from) && !tx.OccurredAt.After(now) && !isCurrent(tx, exceptID) {
			txs = append(txs, tx)
		}
	}
	// От новых к старым: первыми в списке окажутся самые свежие выплаты.
	slices.SortFunc(txs, func(a, b domain.ClassifiedTransaction) int { return b.OccurredAt.Compare(a.OccurredAt) })

	prev := map[domain.IncomeKind][]domain.Money{}
	for _, tx := range txs {
		if len(prev[tx.IncomeKind]) < previousCount {
			prev[tx.IncomeKind] = append(prev[tx.IncomeKind], tx.Amount)
		}
	}
	return prev
}

// isCurrent — это то поступление, по которому решаем. Пустой id — текущего нет.
func isCurrent(tx domain.ClassifiedTransaction, exceptID string) bool {
	return exceptID != "" && tx.ID == exceptID
}
