package config

import (
	"os"
	"strconv"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Engine — пороги движка. Живут в конфиге, а не в коде.
type Engine struct {
	DropThreshold  float64      // падение дохода, при котором спрашиваем клиента
	RaiseThreshold float64      // рост дохода, при котором предлагаем поднять процент
	Buffer         domain.Money // запас, который не трогаем при расчёте безопасной суммы
	AskTTL         time.Duration
}

// Onboarding — правила анкеты при подключении. Баллы — предложение бэкенда, согласуются с аналитиком.
type Onboarding struct {
	MinAge     int // младше — счёт недоступен
	MinScore   int // порог баллов для открытия счёта, из 100
	IncomeTier []IncomeTier

	TypePoints      map[domain.IncomeType]int
	StablePoints    int
	IrregularPoints int

	MonthsLong       int // столько месяцев дохода и больше — «давно»
	MonthsLongPoints int
	MonthsMid        int
	MonthsMidPoints  int

	// Штраф за долговую нагрузку: доля обязательных платежей от дохода, в процентах.
	LoadMidPct      int // больше — штраф LoadMidPenalty
	LoadMidPenalty  int
	LoadHighPct     int // больше — штраф LoadHighPenalty
	LoadHighPenalty int
}

// IncomeTier — «доход от From в месяц даёт Points баллов». Список от большего порога к меньшему.
type IncomeTier struct {
	From   domain.Money
	Points int
}

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	CORSOrigin  string
	Engine      Engine
	Onboarding  Onboarding
}

// DefaultOnboarding — баллы анкеты по умолчанию: доход до 40, тип дохода до 20,
// стабильность до 20, стаж дохода до 20, минус штраф за долговую нагрузку. Порог — 40 из 100.
func DefaultOnboarding() Onboarding {
	return Onboarding{
		MinAge:   18,
		MinScore: 40,
		IncomeTier: []IncomeTier{
			{From: domain.FromRubles(30000), Points: 40},
			{From: domain.FromRubles(15000), Points: 25},
			{From: domain.FromRubles(5000), Points: 10},
		},
		TypePoints: map[domain.IncomeType]int{
			domain.IncomeTypeSalary:      20,
			domain.IncomeTypeScholarship: 15,
			domain.IncomeTypeMixed:       15,
			domain.IncomeTypeFreelance:   10,
		},
		StablePoints:     20,
		IrregularPoints:  5,
		MonthsLong:       6,
		MonthsLongPoints: 20,
		MonthsMid:        3,
		MonthsMidPoints:  10,
		LoadMidPct:       30,
		LoadMidPenalty:   10,
		LoadHighPct:      50,
		LoadHighPenalty:  20,
	}
}

// Load читает настройки из переменных окружения; для незаданных берёт значения по умолчанию.
func Load() Config {
	return Config{
		DatabaseURL: env("DATABASE_URL", "postgres://zapas:zapas@localhost:5433/zapas?sslmode=disable"),
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		CORSOrigin:  env("CORS_ORIGIN", "http://localhost:5173"),
		Engine: Engine{
			DropThreshold:  envFloat("DROP_THRESHOLD", 0.15),
			RaiseThreshold: envFloat("RAISE_THRESHOLD", 0.15),
			Buffer:         domain.FromRubles(int64(envFloat("BUFFER_RUB", 0))),
			AskTTL:         24 * time.Hour,
		},
		Onboarding: onboardingFromEnv(),
	}
}

func onboardingFromEnv() Onboarding {
	o := DefaultOnboarding()
	o.MinAge = envInt("ONBOARDING_MIN_AGE", o.MinAge)
	o.MinScore = envInt("ONBOARDING_MIN_SCORE", o.MinScore)
	return o
}

func envInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return def
	}
	return v
}
