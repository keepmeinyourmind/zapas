package api

import (
	"net/http"
	"strconv"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
)

func (s *Server) putRule(w http.ResponseWriter, r *http.Request) {
	var rule domain.SavingsRule
	if err := decode(r, &rule); err != nil {
		s.writeError(w, err)
		return
	}
	rule.ClientID = r.PathValue("id") // id берём из пути, не из тела
	saved, err := s.svc.PutRule(r.Context(), rule)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) patchRule(w http.ResponseWriter, r *http.Request) {
	var p service.RulePatch
	if err := decode(r, &p); err != nil {
		s.writeError(w, err)
		return
	}
	rule, err := s.svc.PatchRule(r.Context(), r.PathValue("id"), p)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) postTransaction(w http.ResponseWriter, r *http.Request) {
	var tx domain.Transaction
	if err := decode(r, &tx); err != nil {
		s.writeError(w, err)
		return
	}
	res, created, err := s.svc.ProcessTransaction(r.Context(), tx)
	if err != nil {
		s.writeError(w, err)
		return
	}
	status := http.StatusOK // повтор
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, res)
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			s.writeError(w, domain.ValidationError("limit должен быть от 1 до 200"))
			return
		}
		limit = n
	}
	list, err := s.svc.ListTransactions(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
