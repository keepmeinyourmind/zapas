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

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	CORSOrigin  string
	Engine      Engine
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
	}
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
