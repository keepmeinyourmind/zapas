package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

type depositRequest struct {
	Amount domain.Money `json:"amount"`
}

func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	var req depositRequest
	if err := decode(r, &req); err != nil {
		s.writeError(w, err)
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Сумма должна быть положительной"})
		return
	}

	clientID := r.PathValue("id")
	id := fmt.Sprintf("dep_%d", time.Now().UnixNano())

	_, _, err := s.svc.Deposit(r.Context(), clientID, id, req.Amount)
	if err != nil {
		s.writeError(w, err)
		return
	}

	savings, err := s.svc.GetSavings(r.Context(), clientID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, savings)
}
