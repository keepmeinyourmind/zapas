// Package onboarding решает по анкете, можно ли подключить клиенту копилку.
// Чистые функции: без БД и без time.Now(), текущее время передаётся параметром.
package onboarding

import (
	"fmt"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Validate проверяет, что анкета заполнена корректно. Ошибка — это 400, анкета не сохраняется.
func Validate(f domain.OnboardingForm, now time.Time) error {
	switch {
	case f.ClientID == "":
		return domain.ValidationError("Не указан клиент")
	case !f.DataConsent:
		return domain.ValidationError("Нужно согласие на обработку персональных данных")
	case f.BirthDate.IsZero() || f.BirthDate.After(now):
		return domain.ValidationError("Некорректная дата рождения")
	case f.MonthlyIncome < 0 || f.MonthlyPayments < 0 || f.IncomeMonths < 0:
		return domain.ValidationError("Суммы и сроки не могут быть отрицательными")
	}
	switch f.IncomeType {
	case domain.IncomeTypeSalary, domain.IncomeTypeScholarship, domain.IncomeTypeFreelance,
		domain.IncomeTypeMixed, domain.IncomeTypeNone:
	default:
		return domain.ValidationError("Неизвестный тип дохода")
	}
	switch f.IncomeStability {
	case domain.StabilityStable, domain.StabilityIrregular:
	default:
		return domain.ValidationError("Неизвестная стабильность дохода")
	}
	return nil
}

// Evaluate решает по корректной анкете, доступна ли копилка.
// Порядок: возраст → есть ли доход → платежи меньше дохода → баллы.
func Evaluate(f domain.OnboardingForm, now time.Time, cfg config.Onboarding) domain.OnboardingResult {
	res := domain.OnboardingResult{ClientID: f.ClientID, Mode: domain.ModeSmart, Reasons: []string{}}
	if !f.AnalysisConsent {
		res.Mode = domain.ModeSimple
	}

	if age := Age(f.BirthDate, now); age < cfg.MinAge {
		res.Reasons = append(res.Reasons, fmt.Sprintf("Копилка доступна с %d лет", cfg.MinAge))
		return res
	}
	if f.IncomeType == domain.IncomeTypeNone || f.MonthlyIncome == 0 {
		res.Reasons = append(res.Reasons, "Нет дохода, с которого можно откладывать")
		return res
	}
	if f.MonthlyPayments >= f.MonthlyIncome {
		res.Reasons = append(res.Reasons, "Весь доход уходит на обязательные платежи, откладывать сейчас небезопасно")
		return res
	}

	res.Score = Score(f, cfg)
	if res.Score < cfg.MinScore {
		res.Reasons = append(res.Reasons, fmt.Sprintf(
			"Пока не можем подключить копилку: доход нестабилен или большая долговая нагрузка (%d из %d баллов)",
			res.Score, cfg.MinScore))
		return res
	}
	res.Eligible = true
	return res
}

// Score — баллы анкеты от 0 до 100.
func Score(f domain.OnboardingForm, cfg config.Onboarding) int {
	score := 0

	for _, tier := range cfg.IncomeTier { // от большего порога к меньшему: берём первый подходящий
		if f.MonthlyIncome >= tier.From {
			score += tier.Points
			break
		}
	}

	score += cfg.TypePoints[f.IncomeType] // для неизвестного типа мапа вернёт 0

	if f.IncomeStability == domain.StabilityStable {
		score += cfg.StablePoints
	} else {
		score += cfg.IrregularPoints
	}

	switch {
	case f.IncomeMonths >= cfg.MonthsLong:
		score += cfg.MonthsLongPoints
	case f.IncomeMonths >= cfg.MonthsMid:
		score += cfg.MonthsMidPoints
	}

	// Долговая нагрузка в процентах без float: payments/income > pct/100 ⇔ payments*100 > income*pct.
	switch {
	case f.MonthlyPayments*100 > f.MonthlyIncome*domain.Money(cfg.LoadHighPct):
		score -= cfg.LoadHighPenalty
	case f.MonthlyPayments*100 > f.MonthlyIncome*domain.Money(cfg.LoadMidPct):
		score -= cfg.LoadMidPenalty
	}

	return max(score, 0)
}

// Age — полных лет на дату now. День рождения считается наступившим в сам день.
func Age(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}
