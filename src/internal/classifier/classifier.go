package classifier

import "github.com/keepmeinyourmind/zapas/src/internal/domain"

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

	// 4. Выплата от организации - зарплата
	if tx.CounterpartyKind == domain.CounterpartyLegal {
		return domain.IncomeSalary
	}

	return domain.IncomeNone
}
