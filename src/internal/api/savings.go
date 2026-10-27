package api

import (
	"net/http"

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
