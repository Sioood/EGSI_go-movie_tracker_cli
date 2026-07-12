package syncdto

import (
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
)

// Payload is the bulk sync export/import body shared by server and client.
type Payload struct {
	Movies          []domain.Movie      `json:"movies"`
	WatchEntries    []domain.WatchEntry `json:"watch_entries"`
	DeletedMovieIDs []string            `json:"deleted_movie_ids"`
	SourceDeviceID  string              `json:"source_device_id,omitempty"`
	SyncedAt        time.Time           `json:"synced_at"`
}

// ImportResult is returned by POST /api/v1/sync.
type ImportResult struct {
	SyncedMovies       int `json:"synced_movies"`
	SyncedWatchEntries int `json:"synced_watch_entries"`
	DeletedMovies      int `json:"deleted_movies"`
}
