package api

import (
	"net/http"
	"strconv"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string       `json:"id"`
		Amount domain.Money `json:"amount"`
	}
	if err := decode(r, &req); err != nil {
		s.writeError(w, err)
		return
	}
	sv, created, err := s.svc.Withdraw(r.Context(), r.PathValue("id"), req.ID, req.Amount)
	if err != nil {
		s.writeError(w, err)
		return
	}
	status := http.StatusOK // повтор
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, sv)
}

func (s *Server) listRecurring(w http.ResponseWriter, r *http.Request) {
	list, err := s.svc.ListRecurring(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) setPaymentExcluded(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Counterparty string `json:"counterparty"`
		Excluded     bool   `json:"excluded"`
	}
	if err := decode(r, &req); err != nil {
		s.writeError(w, err)
		return
	}
	list, err := s.svc.SetPaymentExcluded(r.Context(), r.PathValue("id"), req.Counterparty, req.Excluded)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// listOperations — GET /api/clients/{id}/operations?limit=50
// Единая история движений копилки: переводы и снятия, новые первыми.
func (s *Server) listOperations(w http.ResponseWriter, r *http.Request) {
	clientID := r.PathValue("id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n < 1 || n > 100 {
			s.writeError(w, domain.ValidationError("limit должен быть от 1 до 100"))
			return
		}
		limit = n
	}
	ops, err := s.svc.ListOperations(r.Context(), clientID, limit)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ops)
}
