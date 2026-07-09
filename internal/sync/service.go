package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/service"
)

const LocalUserID = "local-user"

// SessionAccess holds tokens required for remote sync.
type SessionAccess struct {
	AccessToken  string
	RefreshToken string
	ServerUserID string
}

// TokenRefresher refreshes an access token when sync gets 401.
type TokenRefresher interface {
	Refresh(ctx context.Context, token string) (accessToken, newRefreshToken string, err error)
}

// Service orchestrates push/pull sync between local SQLite and the server.
type Service struct {
	movies      *service.MovieService
	syncRepo    *repository.SyncRepository
	syncClient  *client.SyncClient
	auth        TokenRefresher
	getSession  func() SessionAccess
	getDeviceID func() string
	isOnline    func() bool
	onTokens    func(access, refresh string)
}

// NewService creates a sync orchestrator.
func NewService(
	movies *service.MovieService,
	syncRepo *repository.SyncRepository,
	syncClient *client.SyncClient,
	auth TokenRefresher,
	getSession func() SessionAccess,
	getDeviceID func() string,
	isOnline func() bool,
	onTokens func(access, refresh string),
) *Service {
	return &Service{
		movies:      movies,
		syncRepo:    syncRepo,
		syncClient:  syncClient,
		auth:        auth,
		getSession:  getSession,
		getDeviceID: getDeviceID,
		isOnline:    isOnline,
		onTokens:    onTokens,
	}
}

// Result summarizes a completed sync run.
type Result struct {
	PushedMovies       int
	PushedWatchEntries int
	PulledMovies       int
	PulledWatchEntries int
	DeletedMovies      int
	PendingCount       int
	ConflictCount      int
}

// Run performs push then pull when online and authenticated.
func (s *Service) Run(ctx context.Context) (Result, error) {
	if s.isOnline != nil && !s.isOnline() {
		return Result{}, apperrors.ErrNetwork
	}

	session := s.getSession()
	if session.AccessToken == "" || session.RefreshToken == "" || session.ServerUserID == "" {
		return Result{}, apperrors.ErrUnauthorized
	}

	var result Result
	err := WithRetry(ctx, func() error {
		access := session.AccessToken
		pushPayload, pushedIDs, deletedIDs, err := s.buildPushPayload(ctx, session.ServerUserID)
		if err != nil {
			return err
		}

		importResult, err := s.syncClient.Import(ctx, access, pushPayload)
		if errors.Is(err, apperrors.ErrUnauthorized) {
			var refreshErr error
			access, refreshErr = s.refreshAccess(ctx, &session)
			if refreshErr != nil {
				return refreshErr
			}
			importResult, err = s.syncClient.Import(ctx, access, pushPayload)
		}
		if err != nil {
			return err
		}

		result.PushedMovies = importResult.SyncedMovies
		result.DeletedMovies = importResult.DeletedMovies
		result.PushedWatchEntries = importResult.SyncedWatchEntries

		for _, id := range pushedIDs {
			_ = s.syncRepo.ClearPending(ctx, repository.PendingEntityMovie, id)
		}
		for _, id := range deletedIDs {
			_ = s.syncRepo.ClearPending(ctx, repository.PendingEntityDelete, id)
		}
		for _, entry := range pushPayload.WatchEntries {
			_ = s.syncRepo.ClearPending(ctx, repository.PendingEntityWatchEntry, entry.MovieID)
		}

		remote, err := s.syncClient.Export(ctx, access)
		if errors.Is(err, apperrors.ErrUnauthorized) {
			var refreshErr error
			access, refreshErr = s.refreshAccess(ctx, &session)
			if refreshErr != nil {
				return refreshErr
			}
			remote, err = s.syncClient.Export(ctx, access)
		}
		if err != nil {
			return err
		}

		pullMeta, err := s.syncRepo.GetMetadata(ctx)
		if err != nil {
			return err
		}
		if remote.SourceDeviceID != "" {
			_ = s.syncRepo.UpsertDevice(ctx, remote.SourceDeviceID, "")
		}

		for _, movie := range remote.Movies {
			pendingDelete, err := s.syncRepo.HasPendingDelete(ctx, movie.ID)
			if err != nil {
				return err
			}
			if pendingDelete {
				continue
			}
			if err := s.trackDevice(ctx, movie.UpdatedByDevice); err != nil {
				return err
			}
			applied, err := s.applyRemoteMovie(ctx, pullMeta, movie)
			if err != nil {
				return err
			}
			if applied {
				result.PulledMovies++
			}
		}

		for _, entry := range remote.WatchEntries {
			pendingDelete, err := s.syncRepo.HasPendingDelete(ctx, entry.MovieID)
			if err != nil {
				return err
			}
			if pendingDelete {
				continue
			}
			if err := s.trackDevice(ctx, entry.UpdatedByDevice); err != nil {
				return err
			}
			applied, err := s.applyRemoteWatchEntry(ctx, pullMeta, entry)
			if err != nil {
				return err
			}
			if applied {
				result.PulledWatchEntries++
			}
		}

		meta, err := s.syncRepo.GetMetadata(ctx)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		meta.LastSyncAt = &now
		meta.LastPushAt = &now
		meta.LastPullAt = &now
		if !meta.UserIDMigrated {
			if err := s.syncRepo.MigrateUserID(ctx, LocalUserID, session.ServerUserID); err != nil {
				return err
			}
			meta.UserIDMigrated = true
		}
		if err := s.syncRepo.UpdateMetadata(ctx, meta); err != nil {
			return err
		}

		pending, err := s.syncRepo.PendingCount(ctx)
		if err != nil {
			return err
		}
		result.PendingCount = pending
		conflicts, err := s.syncRepo.ConflictCount(ctx)
		if err != nil {
			return err
		}
		result.ConflictCount = conflicts
		return nil
	})
	if err != nil {
		pending, _ := s.syncRepo.PendingCount(ctx)
		result.PendingCount = pending
		conflicts, _ := s.syncRepo.ConflictCount(ctx)
		result.ConflictCount = conflicts
		return result, err
	}
	return result, nil
}

// PendingCount returns the number of local changes waiting to be pushed.
func (s *Service) PendingCount(ctx context.Context) (int, error) {
	return s.syncRepo.PendingCount(ctx)
}

// ConflictCount returns unresolved sync conflicts.
func (s *Service) ConflictCount(ctx context.Context) (int, error) {
	return s.syncRepo.ConflictCount(ctx)
}

// ListConflicts returns unresolved sync conflicts.
func (s *Service) ListConflicts(ctx context.Context) ([]domain.SyncConflict, error) {
	return s.syncRepo.ListConflicts(ctx)
}

// ResolveConflict applies the chosen version for a conflict.
func (s *Service) ResolveConflict(ctx context.Context, id, choice string) error {
	return s.syncRepo.ResolveConflict(ctx, id, choice)
}

// GetDeviceName returns a friendly device label.
func (s *Service) GetDeviceName(ctx context.Context, deviceID string) (string, error) {
	return s.syncRepo.GetDeviceName(ctx, deviceID)
}

// UserID returns the effective local user id for reads and writes.
func (s *Service) UserID(ctx context.Context) (string, error) {
	meta, err := s.syncRepo.GetMetadata(ctx)
	if err != nil {
		return LocalUserID, err
	}
	if meta.UserIDMigrated {
		session := s.getSession()
		if session.ServerUserID != "" {
			return session.ServerUserID, nil
		}
	}
	return LocalUserID, nil
}

func (s *Service) buildPushPayload(ctx context.Context, serverUserID string) (client.SyncPayload, []string, []string, error) {
	meta, err := s.syncRepo.GetMetadata(ctx)
	if err != nil {
		return client.SyncPayload{}, nil, nil, err
	}

	pending, err := s.syncRepo.ListPending(ctx)
	if err != nil {
		return client.SyncPayload{}, nil, nil, err
	}

	movieIDs := make(map[string]bool)
	watchMovieIDs := make(map[string]bool)
	var deletedIDs []string

	for _, item := range pending {
		switch item.EntityType {
		case repository.PendingEntityMovie:
			movieIDs[item.EntityID] = true
		case repository.PendingEntityWatchEntry:
			watchMovieIDs[item.EntityID] = true
		case repository.PendingEntityDelete:
			if item.Operation == repository.PendingOpDelete {
				deletedIDs = append(deletedIDs, item.EntityID)
			}
		}
	}

	if !meta.UserIDMigrated {
		localMovies, err := s.movies.ListMovies(ctx, LocalUserID)
		if err != nil {
			return client.SyncPayload{}, nil, nil, err
		}
		for _, movie := range localMovies {
			movieIDs[movie.ID] = true
		}
	}

	var movies []domain.Movie
	for id := range movieIDs {
		movie, err := s.movies.GetMovie(ctx, id)
		if err != nil {
			continue
		}
		movie.UserID = serverUserID
		movie.UpdatedByDevice = s.deviceID()
		movies = append(movies, movie)
	}

	var entries []domain.WatchEntry
	for id := range watchMovieIDs {
		entry, err := s.movies.GetWatchEntry(ctx, id)
		if err != nil {
			continue
		}
		entry.UpdatedByDevice = s.deviceID()
		entries = append(entries, entry)
	}

	if !meta.UserIDMigrated {
		for _, movie := range movies {
			entry, err := s.movies.GetWatchEntry(ctx, movie.ID)
			if err == nil {
				entry.UpdatedByDevice = s.deviceID()
				entries = append(entries, entry)
			}
		}
	}

	pushedIDs := make([]string, 0, len(movieIDs))
	for id := range movieIDs {
		pushedIDs = append(pushedIDs, id)
	}

	return client.SyncPayload{
		Movies:          movies,
		WatchEntries:    entries,
		DeletedMovieIDs: deletedIDs,
		SourceDeviceID:  s.deviceID(),
		SyncedAt:        time.Now().UTC(),
	}, pushedIDs, deletedIDs, nil
}

// MarkMoviePending records a local movie mutation for the next push.
func (s *Service) MarkMoviePending(ctx context.Context, movieID string) error {
	return s.syncRepo.MarkPending(ctx, repository.PendingEntityMovie, movieID, repository.PendingOpUpsert)
}

// MarkWatchEntryPending records a local watch entry mutation for the next push.
func (s *Service) MarkWatchEntryPending(ctx context.Context, movieID string) error {
	return s.syncRepo.MarkPending(ctx, repository.PendingEntityWatchEntry, movieID, repository.PendingOpUpsert)
}

// MarkDeletePending records a local delete for the next push.
func (s *Service) MarkDeletePending(ctx context.Context, movieID string) error {
	return s.syncRepo.MarkPending(ctx, repository.PendingEntityDelete, movieID, repository.PendingOpDelete)
}

// MarkPending wraps entity-specific pending markers.
func (s *Service) MarkPending(ctx context.Context, entityType, entityID, operation string) error {
	return s.syncRepo.MarkPending(ctx, entityType, entityID, operation)
}

func (s *Service) refreshAccess(ctx context.Context, session *SessionAccess) (string, error) {
	newAccess, newRefresh, err := s.auth.Refresh(ctx, session.RefreshToken)
	if err != nil {
		return "", err
	}
	session.AccessToken = newAccess
	session.RefreshToken = newRefresh
	if s.onTokens != nil {
		s.onTokens(newAccess, newRefresh)
	}
	return newAccess, nil
}

func (s *Service) deviceID() string {
	if s.getDeviceID != nil {
		return s.getDeviceID()
	}
	return ""
}

func (s *Service) trackDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return nil
	}
	return s.syncRepo.UpsertDevice(ctx, deviceID, "")
}

func (s *Service) applyRemoteMovie(ctx context.Context, meta repository.SyncMetadata, remote domain.Movie) (bool, error) {
	open, err := s.syncRepo.HasOpenConflict(ctx, domain.SyncEntityMovie, remote.ID)
	if err != nil || open {
		return false, err
	}

	local, err := s.movies.GetMovie(ctx, remote.ID)
	if errors.Is(err, apperrors.ErrMovieNotFound) {
		return s.syncRepo.ApplyMovieLWW(ctx, remote)
	}
	if err != nil {
		return false, err
	}

	if !moviesConflict(local, remote) {
		return s.syncRepo.ApplyMovieLWW(ctx, remote)
	}

	if shouldRecordConflict(ctx, s.syncRepo, meta, domain.SyncEntityMovie, remote.ID, local.UpdatedAt, remote.UpdatedAt) {
		return false, s.recordMovieConflict(ctx, local, remote)
	}
	return s.syncRepo.ApplyMovieLWW(ctx, remote)
}

func (s *Service) applyRemoteWatchEntry(ctx context.Context, meta repository.SyncMetadata, remote domain.WatchEntry) (bool, error) {
	open, err := s.syncRepo.HasOpenConflict(ctx, domain.SyncEntityWatchEntry, remote.MovieID)
	if err != nil || open {
		return false, err
	}

	local, err := s.movies.GetWatchEntry(ctx, remote.MovieID)
	if errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		return s.syncRepo.ApplyWatchEntryLWW(ctx, remote)
	}
	if err != nil {
		return false, err
	}

	if !watchEntriesConflict(local, remote) {
		return s.syncRepo.ApplyWatchEntryLWW(ctx, remote)
	}

	if shouldRecordConflict(ctx, s.syncRepo, meta, domain.SyncEntityWatchEntry, remote.MovieID, local.UpdatedAt, remote.UpdatedAt) {
		return false, s.recordWatchEntryConflict(ctx, local, remote)
	}
	return s.syncRepo.ApplyWatchEntryLWW(ctx, remote)
}

func shouldRecordConflict(
	ctx context.Context,
	syncRepo *repository.SyncRepository,
	meta repository.SyncMetadata,
	entityType, entityID string,
	localUpdated, remoteUpdated time.Time,
) bool {
	hasPending, err := syncRepo.HasPending(ctx, entityType, entityID)
	if err != nil {
		return false
	}
	if hasPending {
		return true
	}
	if meta.LastSyncAt == nil {
		return false
	}
	return localUpdated.After(*meta.LastSyncAt) && remoteUpdated.After(*meta.LastSyncAt)
}

func (s *Service) recordMovieConflict(ctx context.Context, local, remote domain.Movie) error {
	localJSON, err := json.Marshal(local)
	if err != nil {
		return err
	}
	remoteJSON, err := json.Marshal(remote)
	if err != nil {
		return err
	}
	return s.syncRepo.RecordConflict(ctx, domain.SyncConflict{
		EntityType:     domain.SyncEntityMovie,
		EntityID:       local.ID,
		LocalJSON:      string(localJSON),
		RemoteJSON:     string(remoteJSON),
		LocalDeviceID:  local.UpdatedByDevice,
		RemoteDeviceID: remote.UpdatedByDevice,
		DetectedAt:     time.Now().UTC(),
	})
}

func (s *Service) recordWatchEntryConflict(ctx context.Context, local, remote domain.WatchEntry) error {
	localJSON, err := json.Marshal(local)
	if err != nil {
		return err
	}
	remoteJSON, err := json.Marshal(remote)
	if err != nil {
		return err
	}
	return s.syncRepo.RecordConflict(ctx, domain.SyncConflict{
		EntityType:     domain.SyncEntityWatchEntry,
		EntityID:       local.MovieID,
		LocalJSON:      string(localJSON),
		RemoteJSON:     string(remoteJSON),
		LocalDeviceID:  local.UpdatedByDevice,
		RemoteDeviceID: remote.UpdatedByDevice,
		DetectedAt:     time.Now().UTC(),
	})
}

// ErrOffline indicates sync was skipped because the client is offline.
var ErrOffline = fmt.Errorf("%w: offline", apperrors.ErrNetwork)
