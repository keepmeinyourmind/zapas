package onboarding

import (
	"errors"
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

var now = time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// anya — анкета Ани из демо: 30 000 ₽, подработка + стипендия, рассрочка 6 000 ₽ + связь 1 000 ₽.
func anya() domain.OnboardingForm {
	return domain.OnboardingForm{
		ClientID: "anya", DataConsent: true, AnalysisConsent: true,
		BirthDate:     date(2006, time.March, 15),
		MonthlyIncome: domain.FromRubles(30000), IncomeType: domain.IncomeTypeMixed,
		IncomeStability: domain.StabilityIrregular, IncomeMonths: 8,
		HasLoans: true, MonthlyPayments: domain.FromRubles(7000),
	}
}

func TestEvaluate(t *testing.T) {
	cfg := config.DefaultOnboarding()

	tests := []struct {
		name         string
		edit         func(f *domain.OnboardingForm) // как анкета отличается от анкеты Ани
		wantEligible bool
		wantScore    int
		wantMode     domain.Mode
	}{
		{"анкета Ани — 80 баллов, доступно", func(f *domain.OnboardingForm) {}, true, 80, domain.ModeSmart},
		{
			name: "студент только со стипендией — 55 баллов, доступно",
			edit: func(f *domain.OnboardingForm) {
				f.MonthlyIncome, f.IncomeType = domain.FromRubles(5000), domain.IncomeTypeScholarship
				f.IncomeStability, f.IncomeMonths = domain.StabilityStable, 3
				f.HasLoans, f.MonthlyPayments = false, 0
			},
			wantEligible: true, wantScore: 55, wantMode: domain.ModeSmart,
		},
		{"младше 18 лет — недоступно", func(f *domain.OnboardingForm) { f.BirthDate = date(2008, time.October, 27) }, false, 0, domain.ModeSmart},
		{"18 лет исполняется сегодня — доступно", func(f *domain.OnboardingForm) { f.BirthDate = date(2008, time.October, 26) }, true, 80, domain.ModeSmart},
		{"нет дохода — недоступно", func(f *domain.OnboardingForm) { f.IncomeType = domain.IncomeTypeNone }, false, 0, domain.ModeSmart},
		{"доход 0 ₽ — недоступно", func(f *domain.OnboardingForm) { f.MonthlyIncome = 0 }, false, 0, domain.ModeSmart},
		{"платежи равны доходу — недоступно", func(f *domain.OnboardingForm) { f.MonthlyPayments = domain.FromRubles(30000) }, false, 0, domain.ModeSmart},
		{"платежи ровно 30% дохода — без штрафа", func(f *domain.OnboardingForm) { f.MonthlyPayments = domain.FromRubles(9000) }, true, 80, domain.ModeSmart},
		{"платежи 40% дохода — минус 10", func(f *domain.OnboardingForm) { f.MonthlyPayments = domain.FromRubles(12000) }, true, 70, domain.ModeSmart},
		{"платежи 60% дохода — минус 20", func(f *domain.OnboardingForm) { f.MonthlyPayments = domain.FromRubles(18000) }, true, 60, domain.ModeSmart},
		{
			name: "мало баллов — недоступно",
			edit: func(f *domain.OnboardingForm) {
				f.MonthlyIncome, f.IncomeType = domain.FromRubles(6000), domain.IncomeTypeFreelance
				f.IncomeMonths, f.MonthlyPayments = 1, domain.FromRubles(4000)
			},
			wantEligible: false, wantScore: 5, wantMode: domain.ModeSmart, // 10 + 10 + 5 + 0 − 20
		},
		{"нет согласия на анализ — простой режим", func(f *domain.OnboardingForm) { f.AnalysisConsent = false }, true, 80, domain.ModeSimple},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := anya()
			tt.edit(&f)
			got := Evaluate(f, now, cfg)
			if got.Eligible != tt.wantEligible || got.Score != tt.wantScore || got.Mode != tt.wantMode {
				t.Errorf("Evaluate() = eligible %v, score %d, mode %s; want %v, %d, %s (reasons: %v)",
					got.Eligible, got.Score, got.Mode, tt.wantEligible, tt.wantScore, tt.wantMode, got.Reasons)
			}
			if !got.Eligible && len(got.Reasons) == 0 {
				t.Error("при отказе должна быть причина для клиента")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(f *domain.OnboardingForm)
		wantErr bool
	}{
		{"корректная анкета", func(f *domain.OnboardingForm) {}, false},
		{"без согласия на обработку данных", func(f *domain.OnboardingForm) { f.DataConsent = false }, true},
		{"дата рождения в будущем", func(f *domain.OnboardingForm) { f.BirthDate = date(2030, time.January, 1) }, true},
		{"нет даты рождения", func(f *domain.OnboardingForm) { f.BirthDate = time.Time{} }, true},
		{"неизвестный тип дохода", func(f *domain.OnboardingForm) { f.IncomeType = "lottery" }, true},
		{"неизвестная стабильность", func(f *domain.OnboardingForm) { f.IncomeStability = "" }, true},
		{"отрицательные платежи", func(f *domain.OnboardingForm) { f.MonthlyPayments = -1 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := anya()
			tt.edit(&f)
			err := Validate(f, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
			var ve domain.ValidationError
			if err != nil && !errors.As(err, &ve) {
				t.Errorf("ошибка должна быть ValidationError (HTTP 400), а не %T", err)
			}
		})
	}
}

func TestAge(t *testing.T) {
	tests := []struct {
		name  string
		birth time.Time
		want  int
	}{
		{"день рождения сегодня", date(2008, time.October, 26), 18},
		{"день рождения завтра", date(2008, time.October, 27), 17},
		{"день рождения был вчера", date(2008, time.October, 25), 18},
		{"родился 29 февраля", date(2008, time.February, 29), 18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Age(tt.birth, now); got != tt.want {
				t.Errorf("Age(%s) = %d, want %d", tt.birth.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}
