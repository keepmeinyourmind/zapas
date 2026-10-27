package api

import "net/http"

// runScenario — только для демо: сбросить Аню и прогнать сценарий.
func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	res, err := s.scenarios.Run(r.Context(), r.PathValue("name"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
