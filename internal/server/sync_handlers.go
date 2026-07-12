package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/transport/syncdto"
)

type syncHandler struct {
	movies *service.MovieService
}

// GET /api/v1/sync — export the authenticated user's full dataset.
func (h *syncHandler) export(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	items, err := h.movies.ListMoviesWithEntries(r.Context(), claims.UserID)
	if err != nil {
		writeInternalError(w, "sync export", err)
		return
	}

	movies := make([]domain.Movie, 0, len(items))
	entries := make([]domain.WatchEntry, 0, len(items))
	for _, item := range items {
		movies = append(movies, item.Movie)
		if item.WatchEntry != nil {
			entries = append(entries, *item.WatchEntry)
		}
	}

	writeJSON(w, http.StatusOK, syncdto.Payload{
		Movies:       movies,
		WatchEntries: entries,
		SyncedAt:     time.Now().UTC(),
	})
}

// POST /api/v1/sync — import a dataset with deletes, movie upserts, then watch entries.
func (h *syncHandler) importData(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body syncdto.Payload
	if err := decodeJSON(w, r, &body); err != nil {
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
			writeInternalError(w, "sync import delete lookup", err)
			return
		}
		if movie.UserID != claims.UserID {
			continue
		}
		if err := h.movies.DeleteMovie(r.Context(), id); err != nil {
			writeInternalError(w, "sync import delete", err)
			return
		}
		deleted++
	}

	syncedMovieIDs := make(map[string]bool, len(body.Movies))
	syncedMovies := 0
	for _, m := range body.Movies {
		saved, _, err := h.movies.SyncUpsertMovie(r.Context(), claims.UserID, m)
		if err != nil {
			writeInternalError(w, "sync import movie", err)
			return
		}
		syncedMovieIDs[saved.ID] = true
		syncedMovies++
	}

	syncedEntries := 0
	for _, e := range body.WatchEntries {
		if !syncedMovieIDs[e.MovieID] {
			continue
		}
		if _, _, err := h.movies.SyncUpsertWatchEntry(r.Context(), e); err != nil {
			writeInternalError(w, "sync import watch entry", err)
			return
		}
		syncedEntries++
	}

	writeJSON(w, http.StatusOK, syncdto.ImportResult{
		SyncedMovies:       syncedMovies,
		SyncedWatchEntries: syncedEntries,
		DeletedMovies:      deleted,
	})
}
