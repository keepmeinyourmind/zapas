package profile

import (
	"slices"
	"strings"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// UpcomingResult — сколько денег понадобится от now до until.
type UpcomingResult struct {
	Total    domain.Money // Payments + Spending
	Payments domain.Money // обязательные платежи
	Spending domain.Money // повседневные траты
	Nearest  *domain.Due  // ближайший обязательный платёж, nil — если его нет; для текста причины
	Dues     []domain.Due // все обязательные платежи периода по датам — для расшифровки клиенту
	Days     int          // за сколько дней посчитаны повседневные траты
}

// DeclaredPaymentsTitle — как называется в расшифровке сумма платежей из анкеты (холодный старт).
const DeclaredPaymentsTitle = "Обязательные платежи из анкеты"

// Upcoming считает, сколько денег понадобится от now до until (обычно — до следующего дохода):
// регулярные платежи, которые выпадут на этот период, плюс повседневные траты × число дней.
//
// declaredMonthly — обязательные платежи в месяц из анкеты. Используются, только пока по истории
// не найдено ни одного регулярного платежа (холодный старт), пропорционально числу дней.
func Upcoming(recurring []domain.RecurringPayment, declaredMonthly, daily domain.Money, now, until time.Time) UpcomingResult {
	var res UpcomingResult
	if !until.After(now) {
		return res
	}
	days := daysBetween(now, until)
	res.Days = days
	res.Spending = daily * domain.Money(days)

	if len(recurring) == 0 {
		res.Payments = declaredMonthly * domain.Money(days) / 30
		if res.Payments > 0 {
			res.Dues = []domain.Due{{Counterparty: DeclaredPaymentsTitle, Amount: res.Payments}}
		}
		res.Total = res.Payments + res.Spending
		return res
	}

	for _, p := range recurring {
		for due := nextDue(p, now); !due.After(until); due = addMonth(due, p.DayOfMonth) {
			res.Payments += p.Amount
			res.Dues = append(res.Dues, domain.Due{Counterparty: p.Counterparty, Amount: p.Amount, At: due})
			if res.Nearest == nil || due.Before(res.Nearest.At) {
				res.Nearest = &domain.Due{Counterparty: p.Counterparty, Amount: p.Amount, At: due}
			}
		}
	}
	// По датам, при равных датах — по имени: порядок в расшифровке всегда одинаковый.
	slices.SortFunc(res.Dues, func(a, b domain.Due) int {
		if c := a.At.Compare(b.At); c != 0 {
			return c
		}
		return strings.Compare(a.Counterparty, b.Counterparty)
	})
	res.Total = res.Payments + res.Spending
	return res
}

// nextDue — когда ждём следующее списание. Если по графику оно уже должно было пройти,
// но не прошло (платёж задерживается), считаем, что оно случится сейчас: деньги на него нужны.
func nextDue(p domain.RecurringPayment, now time.Time) time.Time {
	due := addMonth(p.LastAt, p.DayOfMonth)
	if due.Before(now) {
		return now
	}
	return due
}

// addMonth — тот же день следующего месяца; если такого дня нет (31 февраля) — последний день месяца.
func addMonth(t time.Time, dayOfMonth int) time.Time {
	firstOfNext := time.Date(t.Year(), t.Month()+1, 1, t.Hour(), t.Minute(), 0, 0, t.Location())
	lastDay := firstOfNext.AddDate(0, 1, -1).Day()
	return time.Date(firstOfNext.Year(), firstOfNext.Month(), min(dayOfMonth, lastDay), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// daysBetween — сколько дней от now до until, неполный день считаем целым.
func daysBetween(now, until time.Time) int {
	d := until.Sub(now)
	days := int(d / day)
	if d%day != 0 {
		days++
	}
	return days
}
