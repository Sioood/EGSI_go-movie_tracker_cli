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
	Movies       []domain.Movie      `json:"movies"`
	WatchEntries []domain.WatchEntry `json:"watch_entries"`
	SyncedAt     time.Time           `json:"synced_at"`
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

// POST /api/v1/sync — import a dataset, upserting movies then watch entries.
// The UserID on every movie is forced to the authenticated user.
// Watch entries are only processed for movies that were successfully synced.
func (h *syncHandler) importData(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body syncPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	// Phase 1: upsert movies — record which IDs were accepted.
	syncedMovieIDs := make(map[string]bool, len(body.Movies))
	for _, m := range body.Movies {
		m.UserID = claims.UserID
		saved, err := syncUpsertMovie(r.Context(), h.movies, m)
		if err == nil {
			syncedMovieIDs[saved.ID] = true
		}
	}

	// Phase 2: upsert watch entries only for the user's movies synced above.
	syncedEntries := 0
	for _, entry := range body.WatchEntries {
		if !syncedMovieIDs[entry.MovieID] {
			continue
		}
		if _, err := h.movies.SaveWatchEntry(r.Context(), entry); err == nil {
			syncedEntries++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced_movies":        len(syncedMovieIDs),
		"synced_watch_entries": syncedEntries,
	})
}

// syncUpsertMovie attempts to update an existing movie; if not found it creates it.
func syncUpsertMovie(ctx context.Context, svc *service.MovieService, m domain.Movie) (domain.Movie, error) {
	saved, err := svc.UpdateMovie(ctx, m)
	if errors.Is(err, apperrors.ErrMovieNotFound) {
		return svc.CreateMovie(ctx, m)
	}
	return saved, err
}

