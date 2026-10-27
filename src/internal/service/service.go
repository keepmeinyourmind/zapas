package service

import (
	"log/slog"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"

	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/engine"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Service склеивает логику и БД: единственное место, где они встречаются.
type Service struct {
	store      *storage.Store
	cfg        engine.Config
	askTTL     time.Duration
	onboarding config.Onboarding
	log        *slog.Logger
	clock      *clock.Clock
	now        func() time.Time // текущее время; в демо его можно зафиксировать и сдвигать
}

func New(store *storage.Store, cfg config.Config, clk *clock.Clock, log *slog.Logger) *Service {
	return &Service{store: store, cfg: cfg.Engine, askTTL: cfg.AskTTL, onboarding: cfg.Onboarding,
		log: log, clock: clk, now: clk.Now}
}
