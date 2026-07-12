package server

import (
	"net/http"
	"strconv"

	"github.com/movietracker/movie-tracker/internal/tmdb"
)

type externalHandler struct {
	tmdb interface {
		SearchMovies(r *http.Request, query string, year int) ([]tmdb.SearchResult, error)
	}
}

// GET /api/v1/search/external?q=inception&year=2010
func (h *externalHandler) search(w http.ResponseWriter, r *http.Request) {
	if h.tmdb == nil {
		writeError(w, http.StatusServiceUnavailable, "recherche TMDB indisponible")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "paramètre q requis")
		return
	}

	year := 0
	if rawYear := r.URL.Query().Get("year"); rawYear != "" {
		parsed, err := strconv.Atoi(rawYear)
		if err != nil {
			writeError(w, http.StatusBadRequest, "année invalide")
			return
		}
		year = parsed
	}

	results, err := h.tmdb.SearchMovies(r, query, year)
	if err != nil {
		writeMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tmdb.SearchResponse{Results: results})
}
