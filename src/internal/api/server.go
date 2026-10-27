package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/scenarios"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
)

type Server struct {
	svc        *service.Service
	scenarios  *scenarios.Runner
	clock      *clock.Clock
	corsOrigin string
	log        *slog.Logger
}

func NewServer(svc *service.Service, runner *scenarios.Runner, clk *clock.Clock, corsOrigin string, log *slog.Logger) *Server {
	return &Server{svc: svc, scenarios: runner, clock: clk, corsOrigin: corsOrigin, log: log}
}

// Routes — все маршруты сервера, обёрнутые в CORS.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("PUT /api/clients/{id}/rule", s.putRule)
	mux.HandleFunc("PATCH /api/clients/{id}/rule", s.patchRule)
	mux.HandleFunc("POST /api/transactions", s.postTransaction)
	mux.HandleFunc("GET /api/clients/{id}/transactions", s.listTransactions)
	mux.HandleFunc("POST /api/clients/{id}/onboarding", s.postOnboarding)
	mux.HandleFunc("POST /api/events/account", s.postAccountEvent)
	mux.HandleFunc("GET /api/clients/{id}/decisions", s.listDecisions)
	mux.HandleFunc("POST /api/decisions/{id}/answer", s.answerDecision)
	mux.HandleFunc("GET /api/clients/{id}/savings", s.getSavings)
	mux.HandleFunc("POST /api/debug/advance-time", s.advanceTime)
	mux.HandleFunc("POST /api/scenarios/{name}/run", s.runScenario)
	return s.cors(mux)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError превращает ошибку сервиса в HTTP-ответ единого формата {"error": "..."}.
func (s *Server) writeError(w http.ResponseWriter, err error) {
	var ve domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ve.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrDecisionClosed), errors.Is(err, domain.ErrOnboardingRequired),
		errors.Is(err, domain.ErrNotEligible), errors.Is(err, domain.ErrAccountArrested):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		s.log.Error("внутренняя ошибка", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Внутренняя ошибка сервера"})
	}
}

// decode читает JSON из тела запроса; неверный JSON → 400.
func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return domain.ValidationError("Некорректный JSON")
	}
	return nil
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
