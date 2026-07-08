package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
)

type syncHandler struct {
	movies *service.MovieService
}

type syncPayload struct {
	Movies          []domain.Movie      `json:"movies"`
	WatchEntries    []domain.WatchEntry `json:"watch_entries"`
	DeletedMovieIDs []string            `json:"deleted_movie_ids"`
	SyncedAt        time.Time           `json:"synced_at"`
}

// GET /api/v1/sync — export the authenticated user's full dataset.
func (h *syncHandler) export(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	movies, err := h.movies.ListMovies(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur interne")
		return
	}

	entries := make([]domain.WatchEntry, 0, len(movies))
	for _, m := range movies {
		entry, err := h.movies.GetWatchEntry(r.Context(), m.ID)
		if err == nil {
			entries = append(entries, entry)
		}
	}

	writeJSON(w, http.StatusOK, syncPayload{
		Movies:       movies,
		WatchEntries: entries,
		SyncedAt:     time.Now().UTC(),
	})
}

// POST /api/v1/sync — import a dataset with deletes, movie upserts, then watch entries.
func (h *syncHandler) importData(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body syncPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	deleted := 0
	for _, id := range body.DeletedMovieIDs {
		movie, err := h.movies.GetMovie(r.Context(), id)
		if errors.Is(err, apperrors.ErrMovieNotFound) {
			continue
		}
		if err != nil {
			continue
		}
		if movie.UserID != claims.UserID {
			continue
		}
		if err := h.movies.DeleteMovie(r.Context(), id); err == nil {
			deleted++
		}
	}

	syncedMovieIDs := make(map[string]bool, len(body.Movies))
	for _, m := range body.Movies {
		saved, _, err := h.movies.SyncUpsertMovie(r.Context(), claims.UserID, m)
		if err == nil {
			syncedMovieIDs[saved.ID] = true
		}
	}

	syncedEntries := 0
	for _, entry := range body.WatchEntries {
		movie, err := h.movies.GetMovie(r.Context(), entry.MovieID)
		if err != nil || movie.UserID != claims.UserID {
			continue
		}
		if _, _, err := h.movies.SyncUpsertWatchEntry(r.Context(), entry); err == nil {
			syncedEntries++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced_movies":        len(syncedMovieIDs),
		"synced_watch_entries": syncedEntries,
		"deleted_movies":       deleted,
	})
}

// syncUpsertMovie is kept for backward compatibility in tests if referenced.
func syncUpsertMovie(ctx context.Context, svc *service.MovieService, ownerID string, m domain.Movie) (domain.Movie, error) {
	saved, _, err := svc.SyncUpsertMovie(ctx, ownerID, m)
	return saved, err
}
