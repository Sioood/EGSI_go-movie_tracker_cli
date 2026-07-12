package server

import (
	"net/http"
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
)

type movieHandler struct {
	movies *service.MovieService
}

type movieWithEntry struct {
	Movie      domain.Movie       `json:"movie"`
	WatchEntry *domain.WatchEntry `json:"watch_entry"`
}

func toMovieWithEntry(item service.MovieWithEntry) movieWithEntry {
	return movieWithEntry{
		Movie:      item.Movie,
		WatchEntry: item.WatchEntry,
	}
}

// GET /api/v1/movies?q=&filter=&sort=
func (h *movieHandler) list(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	q := r.URL.Query()

	filter := domain.MovieFilter(q.Get("filter"))
	if filter == "" {
		filter = domain.MovieFilterAll
	}
	sort := domain.MovieSort(q.Get("sort"))
	if sort == "" {
		sort = domain.MovieSortTitle
	}

	items, err := h.movies.SearchMoviesWithEntries(r.Context(), domain.MovieSearchParams{
		UserID: claims.UserID,
		Query:  q.Get("q"),
		Filter: filter,
		Sort:   sort,
	})
	if err != nil {
		writeMovieError(w, err)
		return
	}

	result := make([]movieWithEntry, 0, len(items))
	for _, item := range items {
		result = append(result, toMovieWithEntry(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{"movies": result})
}

// POST /api/v1/movies
func (h *movieHandler) create(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body struct {
		Title      string `json:"title"`
		Year       int    `json:"year"`
		ExternalID string `json:"external_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	movie, err := h.movies.CreateMovie(r.Context(), domain.Movie{
		UserID:     claims.UserID,
		Title:      body.Title,
		Year:       body.Year,
		ExternalID: body.ExternalID,
	})
	if err != nil {
		writeMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, movieWithEntry{Movie: movie})
}

// GET /api/v1/movies/{id}
func (h *movieHandler) get(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	id := r.PathValue("id")

	item, err := h.movies.GetMovieWithEntry(r.Context(), claims.UserID, id)
	if err != nil {
		writeMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMovieWithEntry(item))
}

// PUT /api/v1/movies/{id}
func (h *movieHandler) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	id := r.PathValue("id")

	if _, err := h.movies.GetMovieForUser(r.Context(), claims.UserID, id); err != nil {
		writeMovieError(w, err)
		return
	}

	var body struct {
		Title      string `json:"title"`
		Year       int    `json:"year"`
		ExternalID string `json:"external_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	updated, err := h.movies.UpdateMovie(r.Context(), domain.Movie{
		ID:         id,
		UserID:     claims.UserID,
		Title:      body.Title,
		Year:       body.Year,
		ExternalID: body.ExternalID,
	})
	if err != nil {
		writeMovieError(w, err)
		return
	}

	item := service.MovieWithEntry{Movie: updated}
	entry, err := h.movies.GetWatchEntry(r.Context(), id)
	if err == nil {
		item.WatchEntry = &entry
	}

	writeJSON(w, http.StatusOK, toMovieWithEntry(item))
}

// DELETE /api/v1/movies/{id}
func (h *movieHandler) delete(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	id := r.PathValue("id")

	if _, err := h.movies.GetMovieForUser(r.Context(), claims.UserID, id); err != nil {
		writeMovieError(w, err)
		return
	}

	if err := h.movies.DeleteMovie(r.Context(), id); err != nil {
		writeMovieError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/v1/movies/{id}/watch
func (h *movieHandler) watch(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	id := r.PathValue("id")

	movie, err := h.movies.GetMovieForUser(r.Context(), claims.UserID, id)
	if err != nil {
		writeMovieError(w, err)
		return
	}

	var body struct {
		Watched     bool     `json:"watched"`
		Rating      *float64 `json:"rating"`
		RatingScale int      `json:"rating_scale"`
		Review      string   `json:"review"`
		WatchedAt   string   `json:"watched_at"` // YYYY-MM-DD or ""
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	var watchedAt *time.Time
	if body.WatchedAt != "" {
		t, err := time.Parse("2006-01-02", body.WatchedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "format date invalide (YYYY-MM-DD)")
			return
		}
		watchedAt = &t
	}

	ratingScale := body.RatingScale
	if ratingScale == 0 {
		ratingScale = 10
	}

	existing, _ := h.movies.GetWatchEntry(r.Context(), id)

	entry, err := h.movies.SaveWatchEntry(r.Context(), domain.WatchEntry{
		ID:          existing.ID,
		MovieID:     id,
		Watched:     body.Watched,
		Rating:      body.Rating,
		RatingScale: ratingScale,
		Review:      body.Review,
		WatchedAt:   watchedAt,
	})
	if err != nil {
		writeMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, movieWithEntry{Movie: movie, WatchEntry: &entry})
}
