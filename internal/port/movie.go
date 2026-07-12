package port

import (
	"context"

	"github.com/movietracker/movie-tracker/internal/domain"
)

// MovieOperations is the shared movie + watch-entry API surface.
type MovieOperations interface {
	CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	GetMovie(ctx context.Context, id string) (domain.Movie, error)
	ListMovies(ctx context.Context, userID string) ([]domain.Movie, error)
	SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error)
	UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	DeleteMovie(ctx context.Context, id string) error
	SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error)
	GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error)
}

// StatsOperations reads aggregate statistics for a user.
type StatsOperations interface {
	GetStats(ctx context.Context, userID string) (domain.Stats, error)
}
