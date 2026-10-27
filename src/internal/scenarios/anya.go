// Package scenarios — демо-сценарии клиентки Ани. Детерминированные: одинаковый сценарий
// всегда даёт одинаковое решение. Каждый сценарий = история за полгода + одно ключевое событие.
package scenarios

import (
	"fmt"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// ClientID — героиня демо.
const ClientID = "anya"

// EventAt — когда происходит ключевое событие: 20 октября 2026, 10:00 по Москве (не тихие часы).
// До рассрочки 25-го — 5 дней.
var EventAt = time.Date(2026, time.October, 20, 7, 0, 0, 0, time.UTC)

const (
	employer   = "ООО Ромашка" // подработка Ани
	rassrochka = "Рассрочка за ноутбук"
)

// Scenario — история Ани и ключевое событие.
type Scenario struct {
	Name        string
	Description string
	History     []domain.ClassifiedTransaction // уже размеченная история: так сценарий не зависит от классификатора
	Card        domain.Money                   // на карте до события
	Savings     domain.Money                   // в копилке до события
	Onboarding  domain.OnboardingForm
	Event       domain.Transaction
}

// Опорные числа (на них настроено видео):
//   - доход 30 000 ₽ 20-го числа, рассрочка 6 000 ₽ 25-го, связь 500 ₽ 10-го, подписка 500 ₽ 3-го;
//   - повседневные траты 600 ₽ в день (продукты 4 500 ₽ в неделю);
//   - до следующего дохода (30 дней) понадобится 6 000 + 500 + 500 + 600 × 30 = 25 000 ₽;
//   - на карте до события 2 000 ₽.
//
// income_drop: пришло 24 000 ₽ → на карте 26 000 ₽ → безопасно 26 000 − 25 000 = 1 000 ₽.
// Обычная копилка списала бы 10% = 2 400 ₽, и на рассрочку не хватило бы 1 400 ₽.
var all = map[string]Scenario{
	"stable": {
		Name: "stable", Description: "Пришла обычная зарплата 30 000 ₽",
		History: anyaHistory(30000), Card: domain.FromRubles(2000), Savings: domain.FromRubles(15000),
		Onboarding: anyaForm(), Event: salary("stable", 30000),
	},
	"income_drop": {
		Name: "income_drop", Description: "Пришло 24 000 ₽ вместо 30 000, через 5 дней рассрочка",
		History: anyaHistory(30000), Card: domain.FromRubles(2000), Savings: domain.FromRubles(15000),
		Onboarding: anyaForm(), Event: salary("income_drop", 24000),
	},
	"self_transfer": {
		Name: "self_transfer", Description: "10 000 ₽ со своей карты другого банка — это не доход",
		History: anyaHistory(30000), Card: domain.FromRubles(2000), Savings: domain.FromRubles(15000),
		Onboarding: anyaForm(),
		Event: domain.Transaction{
			ID: "self_transfer-event", Amount: domain.FromRubles(10000), Type: domain.TxKindIncoming,
			Counterparty: "Моя карта Т-Банка", CounterpartyKind: domain.CounterpartySelf, OccurredAt: EventAt,
		},
	},
	"new_client": {
		Name: "new_client", Description: "Первая зарплата нового клиента: истории ещё нет",
		History: nil, Card: domain.FromRubles(2000), Savings: 0,
		Onboarding: anyaForm(), Event: salary("new_client", 30000),
	},
	"income_rise": {
		Name: "income_rise", Description: "Доход вырос: пришло 36 000 ₽ вместо 30 000",
		History: anyaHistory(30000), Card: domain.FromRubles(2000), Savings: domain.FromRubles(15000),
		Onboarding: anyaForm(), Event: salary("income_rise", 36000),
	},
}

// Get — сценарий по имени.
func Get(name string) (Scenario, bool) {
	s, ok := all[name]
	return s, ok
}

// Names — имена всех сценариев в порядке показа.
func Names() []string {
	return []string{"stable", "income_drop", "self_transfer", "new_client", "income_rise"}
}

func salary(scenario string, rubles int64) domain.Transaction {
	return domain.Transaction{
		ID: scenario + "-event", Amount: domain.FromRubles(rubles), Type: domain.TxKindIncoming,
		Counterparty: employer, CounterpartyKind: domain.CounterpartyLegal, OccurredAt: EventAt,
	}
}

func anyaForm() domain.OnboardingForm {
	return domain.OnboardingForm{
		ClientID: ClientID, DataConsent: true, AnalysisConsent: true,
		BirthDate:     time.Date(2006, time.March, 15, 0, 0, 0, 0, time.UTC),
		MonthlyIncome: domain.FromRubles(30000), IncomeType: domain.IncomeTypeMixed,
		IncomeStability: domain.StabilityIrregular, IncomeMonths: 8,
		HasLoans: true, MonthlyPayments: domain.FromRubles(7000),
	}
}

// anyaHistory — полгода операций Ани до события: доход 20-го с мая по сентябрь,
// рассрочка, связь, подписка и продукты.
func anyaHistory(monthly int64) []domain.ClassifiedTransaction {
	var h []domain.ClassifiedTransaction
	n := 0
	add := func(kind domain.IncomeKind, typ domain.TxKind, who string, cpKind domain.CounterpartyKind, rubles int64, at time.Time) {
		n++
		h = append(h, domain.ClassifiedTransaction{
			Transaction: domain.Transaction{
				ID: fmt.Sprintf("hist-%03d", n), ClientID: ClientID, Amount: domain.FromRubles(rubles), Type: typ,
				Counterparty: who, CounterpartyKind: cpKind, OccurredAt: at,
			},
			IncomeKind: kind,
		})
	}
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 7, 0, 0, 0, time.UTC) }
	out := domain.TxKindOutgoing

	for m := time.May; m <= time.September; m++ {
		add(domain.IncomeSalary, domain.TxKindIncoming, employer, domain.CounterpartyLegal, monthly, day(m, 20))
		add(domain.IncomeNone, out, rassrochka, domain.CounterpartyLegal, 6000, day(m, 25))
	}
	for m := time.May; m <= time.October; m++ {
		add(domain.IncomeNone, out, "Яндекс Плюс", domain.CounterpartyLegal, 500, day(m, 3))
		add(domain.IncomeNone, out, "МТС", domain.CounterpartyLegal, 500, day(m, 10))
	}
	// Продукты раз в неделю. В последние 30 дней до события — ровно 4 раза по 4 500 ₽ = 18 000 ₽,
	// то есть 600 ₽ в день.
	for d := day(time.May, 5); d.Before(EventAt); d = d.AddDate(0, 0, 7) {
		add(domain.IncomeNone, out, "Пятёрочка", domain.CounterpartyLegal, 4500, d)
	}
	return h
}
