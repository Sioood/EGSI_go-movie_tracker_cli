package server

import (
	"encoding/json"
	"net/http"

	"github.com/movietracker/movie-tracker/internal/version"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Version: version.Version,
	})
}
