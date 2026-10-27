package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
)

// limitParam читает ?limit=, по умолчанию 50, от 1 до 200.
func limitParam(r *http.Request) (int, error) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return 50, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 200 {
		return 0, domain.ValidationError("limit должен быть от 1 до 200")
	}
	return n, nil
}

func (s *Server) listDecisions(w http.ResponseWriter, r *http.Request) {
	limit, err := limitParam(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	status := domain.DecisionStatus(r.URL.Query().Get("status"))
	list, err := s.svc.ListDecisions(r.Context(), r.PathValue("id"), status, limit)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) answerDecision(w http.ResponseWriter, r *http.Request) {
	if _, err := uuid.Parse(r.PathValue("id")); err != nil {
		s.writeError(w, domain.ErrNotFound) // id решений — UUID; другой строки в базе быть не может
		return
	}
	var a service.Answer
	if err := decode(r, &a); err != nil {
		s.writeError(w, err)
		return
	}
	d, err := s.svc.AnswerDecision(r.Context(), r.PathValue("id"), a)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) getSavings(w http.ResponseWriter, r *http.Request) {
	sv, err := s.svc.GetSavings(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sv)
}

// advanceTime — только для демо: сдвинуть время вперёд, чтобы сработали таймауты вопросов.
func (s *Server) advanceTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hours int `json:"hours"`
	}
	if err := decode(r, &req); err != nil {
		s.writeError(w, err)
		return
	}
	if req.Hours < 1 {
		s.writeError(w, domain.ValidationError("hours должно быть не меньше 1"))
		return
	}
	s.clock.Advance(time.Duration(req.Hours) * time.Hour)
	expired, err := s.svc.ExpireDue(r.Context())
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"now": s.clock.Now(), "expired": expired})
}
