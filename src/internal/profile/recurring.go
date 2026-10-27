package profile

import (
	"slices"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

const (
	day = 24 * time.Hour

	recurringMinCount = 2        // столько раз подряд — уже регулярный платёж
	recurringMinGap   = 25 * day // интервал между списаниями: примерно месяц
	recurringMaxGap   = 35 * day
	recurringTolerPct = 10 // сумма может отличаться от обычной на ±10%
	spendingWindow    = 30 // повседневные траты считаем за последние 30 дней
)

// DetectRecurring находит регулярные списания по истории: один получатель, не меньше 2 раз подряд,
// интервал 25–35 дней, сумма ±10% от обычной. Смотрит на последние списания каждого получателя:
// если клиент перестал платить (последнее списание старше 35 дней) или платёж стал нерегулярным,
// он больше не считается обязательным.
func DetectRecurring(history []domain.Transaction, now time.Time) []domain.RecurringPayment {
	byRecipient := map[string][]domain.Transaction{}
	for _, tx := range history {
		if tx.Type == domain.TxKindOutgoing && tx.Counterparty != "" {
			byRecipient[tx.Counterparty] = append(byRecipient[tx.Counterparty], tx)
		}
	}

	var out []domain.RecurringPayment
	for recipient, txs := range byRecipient {
		slices.SortFunc(txs, func(a, b domain.Transaction) int { return a.OccurredAt.Compare(b.OccurredAt) })

		// Идём от последнего списания назад, пока интервалы похожи на «раз в месяц».
		streak := []domain.Transaction{txs[len(txs)-1]}
		for i := len(txs) - 2; i >= 0; i-- {
			gap := streak[0].OccurredAt.Sub(txs[i].OccurredAt)
			if gap < recurringMinGap || gap > recurringMaxGap {
				break
			}
			streak = append([]domain.Transaction{txs[i]}, streak...)
		}
		if len(streak) < recurringMinCount {
			continue
		}
		if now.Sub(streak[len(streak)-1].OccurredAt) > recurringMaxGap {
			continue // давно не платил — подписку отменили, рассрочку закрыли
		}

		amounts := make([]domain.Money, len(streak))
		for i, tx := range streak {
			amounts[i] = tx.Amount
		}
		typical := domain.Median(amounts)
		if !allWithin(amounts, typical, recurringTolerPct) {
			continue // суммы скачут — это не обязательный платёж, а обычные траты
		}

		last := streak[len(streak)-1]
		out = append(out, domain.RecurringPayment{
			Counterparty: recipient,
			Amount:       typical,
			DayOfMonth:   last.OccurredAt.Day(),
			LastAt:       last.OccurredAt,
		})
	}

	// Мапа обходится в случайном порядке — сортируем, чтобы результат был детерминированным.
	slices.SortFunc(out, func(a, b domain.RecurringPayment) int {
		if a.Counterparty < b.Counterparty {
			return -1
		}
		if a.Counterparty > b.Counterparty {
			return 1
		}
		return 0
	})
	return out
}

// allWithin — все суммы отличаются от typical не больше чем на pct процентов.
func allWithin(amounts []domain.Money, typical domain.Money, pct int) bool {
	for _, a := range amounts {
		diff := a - typical
		if diff < 0 {
			diff = -diff
		}
		if diff*100 > typical*domain.Money(pct) {
			return false
		}
	}
	return true
}

// DailySpending — средние повседневные траты в день за последние 30 дней,
// без регулярных платежей (они учитываются отдельно, по датам).
func DailySpending(history []domain.Transaction, recurring []domain.RecurringPayment, now time.Time) domain.Money {
	isRecurring := map[string]bool{}
	for _, p := range recurring {
		isRecurring[p.Counterparty] = true
	}
	from := now.Add(-spendingWindow * day)

	var total domain.Money
	for _, tx := range history {
		if tx.Type != domain.TxKindOutgoing || isRecurring[tx.Counterparty] {
			continue
		}
		if tx.OccurredAt.After(from) && !tx.OccurredAt.After(now) {
			total += tx.Amount
		}
	}
	return total / spendingWindow
}
