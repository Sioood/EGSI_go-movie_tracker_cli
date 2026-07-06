package server

import (
	"net/http"

	"github.com/movietracker/movie-tracker/internal/service"
)

type statsHandler struct {
	stats *service.StatsService
}

// GET /api/v1/stats
func (h *statsHandler) get(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	s, err := h.stats.GetStats(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur interne")
		return
	}

	writeJSON(w, http.StatusOK, s)
}
