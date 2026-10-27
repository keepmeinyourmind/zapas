package classifier

import (
	"slices"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Classify определяет вид поступления.
//
//	tx        — новая операция
//	history   — прошлые операции клиента, отсортированы по времени
//	confirmed — отправители-физлица, которых клиент подтвердил как источник дохода
func Classify(tx domain.Transaction, history []domain.Transaction, confirmed map[string]bool) domain.IncomeKind {
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
		if len(past) == 0 {
			return domain.IncomeSalary
		}
		if tx.Amount*2 > domain.Median(past)*3 {
			return domain.IncomeBonus
		}
		if tx.Amount*10 < slices.Max(past)*7 {
			return domain.IncomeAdvance
		}
		return domain.IncomeSalary
	}
	return domain.IncomeNone
}

// pastPayments — суммы прошлых выплат того же отправителя за последние 40 дней.
func pastPayments(tx domain.Transaction, history []domain.Transaction) []domain.Money {
	var past []domain.Money
	for _, h := range history {
		if h.Type != domain.TxKindIncoming || h.Counterparty != tx.Counterparty {
			continue
		}
		gap := tx.OccurredAt.Sub(h.OccurredAt)
		if gap <= 0 || gap > 40*24*time.Hour {
			continue
		}
		past = append(past, h.Amount)
	}
	return past
}
