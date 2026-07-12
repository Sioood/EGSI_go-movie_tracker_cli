package sync

import (
	"context"

	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
)

// MovieReader is the subset of movie operations used by the hybrid client.
type MovieReader interface {
	CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	GetMovie(ctx context.Context, id string) (domain.Movie, error)
	ListMovies(ctx context.Context, userID string) ([]domain.Movie, error)
	SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error)
	UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	DeleteMovie(ctx context.Context, id string) error
	SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error)
	GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error)
}

// StatsReader loads statistics for a user.
type StatsReader interface {
	GetStats(ctx context.Context, userID string) (domain.Stats, error)
}

// SyncTrigger runs a sync cycle asynchronously.
type SyncTrigger interface {
	Run(ctx context.Context) (Result, error)
	PendingCount(ctx context.Context) (int, error)
	UserID(ctx context.Context) (string, error)
	MarkMoviePending(ctx context.Context, movieID string) error
	MarkWatchEntryPending(ctx context.Context, movieID string) error
	MarkDeletePending(ctx context.Context, movieID string) error
}

// HybridClient writes locally, marks pending changes, and optionally triggers sync.
type HybridClient struct {
	local       MovieReader
	stats       StatsReader
	sync        SyncTrigger
	userID      func(context.Context) (string, error)
	getDeviceID func() string
	online      func() bool
	onSync      func()
}

// NewHybridClient creates a local-first MovieClient with sync side effects.
func NewHybridClient(local MovieReader, stats StatsReader, syncSvc SyncTrigger, userID func(context.Context) (string, error), getDeviceID func() string, online func() bool, onSync func()) *HybridClient {
	return &HybridClient{
		local:       local,
		stats:       stats,
		sync:        syncSvc,
		userID:      userID,
		getDeviceID: getDeviceID,
		online:      online,
		onSync:      onSync,
	}
}

func (h *HybridClient) resolveUserID(ctx context.Context) (string, error) {
	if h.userID != nil {
		return h.userID(ctx)
	}
	return LocalUserID, nil
}

func (h *HybridClient) currentDeviceID() string {
	if h.getDeviceID != nil {
		return h.getDeviceID()
	}
	return ""
}

func (h *HybridClient) maybeSync() {
	if h.onSync != nil && h.online != nil && h.online() {
		h.onSync()
	}
}

func (h *HybridClient) CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	userID, err := h.resolveUserID(ctx)
	if err != nil {
		return domain.Movie{}, err
	}
	movie.UserID = userID
	movie.UpdatedByDevice = h.currentDeviceID()
	created, err := h.local.CreateMovie(ctx, movie)
	if err != nil {
		return domain.Movie{}, err
	}
	_ = h.sync.MarkMoviePending(ctx, created.ID)
	h.maybeSync()
	return created, nil
}

func (h *HybridClient) GetMovie(ctx context.Context, id string) (domain.Movie, error) {
	return h.local.GetMovie(ctx, id)
}

func (h *HybridClient) ListMovies(ctx context.Context, userID string) ([]domain.Movie, error) {
	return h.local.ListMovies(ctx, userID)
}

func (h *HybridClient) SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	return h.local.SearchMovies(ctx, params)
}

func (h *HybridClient) UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	movie.UpdatedByDevice = h.currentDeviceID()
	updated, err := h.local.UpdateMovie(ctx, movie)
	if err != nil {
		return domain.Movie{}, err
	}
	_ = h.sync.MarkMoviePending(ctx, updated.ID)
	h.maybeSync()
	return updated, nil
}

func (h *HybridClient) DeleteMovie(ctx context.Context, id string) error {
	if err := h.sync.MarkDeletePending(ctx, id); err != nil {
		return err
	}
	if err := h.local.DeleteMovie(ctx, id); err != nil {
		return err
	}
	h.maybeSync()
	return nil
}

func (h *HybridClient) SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	entry.UpdatedByDevice = h.currentDeviceID()
	saved, err := h.local.SaveWatchEntry(ctx, entry)
	if err != nil {
		return domain.WatchEntry{}, err
	}
	_ = h.sync.MarkWatchEntryPending(ctx, saved.MovieID)
	h.maybeSync()
	return saved, nil
}

func (h *HybridClient) GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	return h.local.GetWatchEntry(ctx, movieID)
}

func (h *HybridClient) GetStats(ctx context.Context, userID string) (domain.Stats, error) {
	return h.stats.GetStats(ctx, userID)
}

// LocalService combines movie and stats services for the hybrid wrapper.
type LocalService struct {
	*service.MovieService
	*service.StatsService
}
