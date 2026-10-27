package classifier

import (
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Window — за сколько дней назад смотрим прошлые выплаты того же отправителя.
const Window = 40 * 24 * time.Hour

const (
	// Премия — выплата больше обычной в 1,5 раза: tx*2 > base*3.
	bonusNum, bonusDen = 3, 2
	// Аванс — выплата меньше 70% обычной: tx*10 < base*7.
	advanceNum, advanceDen = 7, 10
	// Две выплаты ближе 25 дней — отправитель платит дважды в месяц.
	// Та же граница отделяет «месяц назад» от «в этом месяце».
	twiceMonthly = 25 * 24 * time.Hour
	// «Премия» месяц назад в пределах ±30% от новой выплаты — это повторяющаяся зарплата.
	repeatPercent = 30
)

// Classify определяет вид поступления.
//
//	tx        — новая операция
//	history   — прошлые операции клиента с метками классификатора, отсортированы по времени
//	confirmed — отправители-физлица, которых клиент подтвердил как источник дохода
func Classify(tx domain.Transaction, history []domain.ClassifiedTransaction, confirmed map[string]bool) domain.IncomeKind {
	// 1. Не входящая операция (перевод из копилки, возврат, кредит, списание) - не доход
	if tx.Type != domain.TxKindIncoming {
		return domain.IncomeNone
	}

	// 2. Свои деньги с другого счета или деньги от банка (кредит, проценты) - не доход
	if tx.CounterpartyKind == domain.CounterpartySelf || tx.CounterpartyKind == domain.CounterpartyBank {
		return domain.IncomeNone
	}

	// 3. Перевод от физлица - доход (фриланс), только если клиент подтвердил отправителя
	if tx.CounterpartyKind == domain.CounterpartyPerson {
		if confirmed[tx.Counterparty] {
			return domain.IncomeFreelance
		}
		return domain.IncomeNone
	}

	// 4. Выплата от организации: зарплата, аванс или премия - по прошлым выплатам того же отправителя
	if tx.CounterpartyKind == domain.CounterpartyLegal {
		past := pastPayments(tx, history)

		// Обычные выплаты — всё, кроме премий: премия не задаёт размер зарплаты
		var regular []domain.ClassifiedTransaction
		for _, p := range past {
			if p.IncomeKind != domain.IncomeBonus {
				regular = append(regular, p)
			}
		}
		if len(regular) == 0 {
			return domain.IncomeSalary
		}
		base := regular[0].Amount
		for _, p := range regular {
			base = max(base, p.Amount)
		}

		if tx.Amount*bonusDen > base*bonusNum && !repeatsBonus(tx, past) {
			return domain.IncomeBonus
		}
		// Меньшая выплата — аванс, только если отправитель платит дважды в месяц.
		// Иначе это зарплата, и падение дохода оценит движок.
		if tx.Amount*advanceDen < base*advanceNum && paysTwiceMonthly(regular) {
			return domain.IncomeAdvance
		}
		return domain.IncomeSalary
	}
	return domain.IncomeNone
}

// pastPayments — прошлые выплаты того же отправителя за Window.
func pastPayments(tx domain.Transaction, history []domain.ClassifiedTransaction) []domain.ClassifiedTransaction {
	var past []domain.ClassifiedTransaction
	for _, h := range history {
		if h.Type != domain.TxKindIncoming || h.Counterparty != tx.Counterparty {
			continue
		}
		gap := tx.OccurredAt.Sub(h.OccurredAt)
		if gap <= 0 || gap > Window {
			continue
		}
		past = append(past, h)
	}
	return past
}

// repeatsBonus — месяц назад была «премия» почти той же суммы (±30%).
// Премия каждый месяц — это на самом деле зарплата.
func repeatsBonus(tx domain.Transaction, past []domain.ClassifiedTransaction) bool {
	for _, p := range past {
		if p.IncomeKind != domain.IncomeBonus || tx.OccurredAt.Sub(p.OccurredAt) < twiceMonthly {
			continue
		}
		diff := p.Amount - tx.Amount
		if diff < 0 {
			diff = -diff
		}
		if diff*100 <= tx.Amount*repeatPercent {
			return true
		}
	}
	return false
}

// paysTwiceMonthly — среди выплат есть две с разницей меньше 25 дней.
func paysTwiceMonthly(regular []domain.ClassifiedTransaction) bool {
	for i := range regular {
		for j := i + 1; j < len(regular); j++ {
			gap := regular[j].OccurredAt.Sub(regular[i].OccurredAt)
			if gap < 0 {
				gap = -gap
			}
			if gap < twiceMonthly {
				return true
			}
		}
	}
	return false
}
