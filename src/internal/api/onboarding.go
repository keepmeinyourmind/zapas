package api

import (
	"net/http"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// onboardingRequest — анкета в JSON. Дата рождения — строка "2006-03-15".
type onboardingRequest struct {
	DataConsent     bool                   `json:"data_consent"`
	AnalysisConsent bool                   `json:"analysis_consent"`
	BirthDate       string                 `json:"birth_date"`
	MonthlyIncome   domain.Money           `json:"monthly_income"`
	IncomeType      domain.IncomeType      `json:"income_type"`
	IncomeStability domain.IncomeStability `json:"income_stability"`
	IncomeMonths    int                    `json:"income_months"`
	HasLoans        bool                   `json:"has_loans"`
	MonthlyPayments domain.Money           `json:"monthly_payments"`
}

func (s *Server) postOnboarding(w http.ResponseWriter, r *http.Request) {
	var req onboardingRequest
	if err := decode(r, &req); err != nil {
		s.writeError(w, err)
		return
	}
	birth, err := time.Parse(time.DateOnly, req.BirthDate)
	if err != nil {
		s.writeError(w, domain.ValidationError("Дата рождения должна быть в формате ГГГГ-ММ-ДД"))
		return
	}
	res, err := s.svc.SubmitOnboarding(r.Context(), domain.OnboardingForm{
		ClientID:        r.PathValue("id"),
		DataConsent:     req.DataConsent,
		AnalysisConsent: req.AnalysisConsent,
		BirthDate:       birth,
		MonthlyIncome:   req.MonthlyIncome,
		IncomeType:      req.IncomeType,
		IncomeStability: req.IncomeStability,
		IncomeMonths:    req.IncomeMonths,
		HasLoans:        req.HasLoans,
		MonthlyPayments: req.MonthlyPayments,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) postAccountEvent(w http.ResponseWriter, r *http.Request) {
	var e domain.AccountEvent
	if err := decode(r, &e); err != nil {
		s.writeError(w, err)
		return
	}
	st, created, err := s.svc.ProcessAccountEvent(r.Context(), e)
	if err != nil {
		s.writeError(w, err)
		return
	}
	status := http.StatusOK // повтор события
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, st)
}
