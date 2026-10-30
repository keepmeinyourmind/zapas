package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // часовые пояса внутри бинарника: в Docker-образе alpine их нет

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/keepmeinyourmind/zapas/src/internal/api"
	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/middleware"
	"github.com/keepmeinyourmind/zapas/src/internal/scenarios"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("сервер остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ctx отменяется по Ctrl+C или SIGTERM (docker stop) — так сервер завершается аккуратно.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	// Миграции при старте: docker compose up — и база готова.
	if err := storage.Migrate(pool); err != nil {
		return err
	}
	log.Info("миграции применены")

	clk := clock.New()
	store := storage.New(pool)
	svc := service.New(store, cfg, clk, log)
	runner := scenarios.NewRunner(store, svc, clk)
	handler := api.NewServer(svc, runner, clk, cfg.CORSOrigin, log)

	// Раз в минуту закрываем вопросы, на которые клиент не ответил вовремя.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if expired, err := svc.ExpireDue(ctx); err != nil {
					log.Error("таймауты вопросов", "err", err)
				} else if len(expired) > 0 {
					log.Info("закрыты вопросы без ответа", "count", len(expired))
				}
			}
		}
	}()

	// Rate Limiting: 100 запросов в минуту на один IP
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// Запускаем фоновую очистку старых записей (каждые 10 минут)
	go rateLimiter.Cleanup(10 * time.Minute)

	// Таймауты: медленный или зависший клиент не держит соединение бесконечно.
	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: rateLimiter.Middleware(
			middleware.SecurityHeaders(
				middleware.ProductionGuard(handler.Routes()),
			),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("слушаем", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("останавливаемся")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
