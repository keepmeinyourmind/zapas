package domain

import "time"

// Mode — режим копилки.
type Mode string

const (
	ModeSmart  Mode = "smart"  // анализ доходов и расходов, безопасная сумма, вопросы клиенту
	ModeSimple Mode = "simple" // клиент отказался от анализа: фиксированный процент, как автопополнение
)

// IncomeType — тип дохода из анкеты (что клиент сам о себе рассказал).
type IncomeType string

const (
	IncomeTypeSalary      IncomeType = "salary"
	IncomeTypeScholarship IncomeType = "scholarship"
	IncomeTypeFreelance   IncomeType = "freelance"
	IncomeTypeMixed       IncomeType = "mixed"
	IncomeTypeNone        IncomeType = "none"
)

// IncomeStability — насколько стабилен доход, по словам клиента.
type IncomeStability string

const (
	StabilityStable    IncomeStability = "stable"
	StabilityIrregular IncomeStability = "irregular"
)

// OnboardingForm — анкета при подключении копилки.
type OnboardingForm struct {
	ClientID        string
	DataConsent     bool // согласие на обработку персональных данных (152-ФЗ)
	AnalysisConsent bool // согласие на анализ доходов и расходов; false → простой режим
	BirthDate       time.Time
	MonthlyIncome   Money
	IncomeType      IncomeType
	IncomeStability IncomeStability
	IncomeMonths    int   // сколько месяцев уже есть доход
	HasLoans        bool  // есть кредиты, рассрочки, займы
	MonthlyPayments Money // обязательные платежи в месяц: кредиты, аренда, связь, подписки
}

// OnboardingResult — итог анкеты.
type OnboardingResult struct {
	ClientID string   `json:"client_id"`
	Eligible bool     `json:"eligible"`
	Score    int      `json:"score"`
	Mode     Mode     `json:"mode"`
	Reasons  []string `json:"reasons"` // почему счёт недоступен; пусто, если доступен
}
