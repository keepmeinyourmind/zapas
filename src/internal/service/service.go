package service

import (
	"log/slog"

	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Service склеивает логику и БД: единственное место, где они встречаются.
type Service struct {
	store *storage.Store
	cfg   config.Engine
	log   *slog.Logger
}

func New(store *storage.Store, cfg config.Engine, log *slog.Logger) *Service {
	return &Service{store: store, cfg: cfg, log: log}
}
