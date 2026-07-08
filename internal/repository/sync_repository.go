package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

const (
	PendingEntityMovie      = "movie"
	PendingEntityWatchEntry = "watch_entry"
	PendingEntityDelete     = "delete"
	PendingOpUpsert         = "upsert"
	PendingOpDelete         = "delete"
)

// SyncMetadata holds global sync state for the client database.
type SyncMetadata struct {
	LastSyncAt      *time.Time
	LastPushAt      *time.Time
	LastPullAt      *time.Time
	UserIDMigrated  bool
}

// PendingChange is a local mutation waiting to be pushed.
type PendingChange struct {
	EntityType string
	EntityID   string
	Operation  string
}

type SyncRepository struct {
	db     *sql.DB
	movies *MovieRepository
	entries *WatchEntryRepository
}

func NewSyncRepository(db *sql.DB, movies *MovieRepository, entries *WatchEntryRepository) *SyncRepository {
	return &SyncRepository{db: db, movies: movies, entries: entries}
}

func (r *SyncRepository) GetMetadata(ctx context.Context) (SyncMetadata, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT last_sync_at, last_push_at, last_pull_at, user_id_migrated
		FROM sync_metadata
		WHERE id = 1
	`)

	var lastSync, lastPush, lastPull sql.NullString
	var migrated int
	if err := row.Scan(&lastSync, &lastPush, &lastPull, &migrated); err != nil {
		return SyncMetadata{}, fmt.Errorf("%w: get sync metadata: %v", apperrors.ErrDB, err)
	}

	meta := SyncMetadata{UserIDMigrated: migrated != 0}
	if lastSync.Valid {
		t, err := parseTime(lastSync.String)
		if err != nil {
			return SyncMetadata{}, err
		}
		meta.LastSyncAt = &t
	}
	if lastPush.Valid {
		t, err := parseTime(lastPush.String)
		if err != nil {
			return SyncMetadata{}, err
		}
		meta.LastPushAt = &t
	}
	if lastPull.Valid {
		t, err := parseTime(lastPull.String)
		if err != nil {
			return SyncMetadata{}, err
		}
		meta.LastPullAt = &t
	}
	return meta, nil
}

func (r *SyncRepository) UpdateMetadata(ctx context.Context, meta SyncMetadata) error {
	var lastSync, lastPush, lastPull any
	if meta.LastSyncAt != nil {
		lastSync = formatTime(*meta.LastSyncAt)
	}
	if meta.LastPushAt != nil {
		lastPush = formatTime(*meta.LastPushAt)
	}
	if meta.LastPullAt != nil {
		lastPull = formatTime(*meta.LastPullAt)
	}
	migrated := 0
	if meta.UserIDMigrated {
		migrated = 1
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_metadata
		SET last_sync_at = ?, last_push_at = ?, last_pull_at = ?, user_id_migrated = ?
		WHERE id = 1
	`, lastSync, lastPush, lastPull, migrated)
	if err != nil {
		return fmt.Errorf("%w: update sync metadata: %v", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) MarkPending(ctx context.Context, entityType, entityID, operation string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_pending (entity_type, entity_id, operation)
		VALUES (?, ?, ?)
		ON CONFLICT(entity_type, entity_id) DO UPDATE SET operation = excluded.operation
	`, entityType, entityID, operation)
	if err != nil {
		return fmt.Errorf("%w: mark pending: %v", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ListPending(ctx context.Context) ([]PendingChange, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT entity_type, entity_id, operation
		FROM sync_pending
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: list pending: %v", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var pending []PendingChange
	for rows.Next() {
		var item PendingChange
		if err := rows.Scan(&item.EntityType, &item.EntityID, &item.Operation); err != nil {
			return nil, fmt.Errorf("%w: scan pending: %v", apperrors.ErrDB, err)
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate pending: %v", apperrors.ErrDB, err)
	}
	return pending, nil
}

func (r *SyncRepository) PendingCount(ctx context.Context) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_pending`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: pending count: %v", apperrors.ErrDB, err)
	}
	return count, nil
}

func (r *SyncRepository) ClearPending(ctx context.Context, entityType, entityID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM sync_pending WHERE entity_type = ? AND entity_id = ?
	`, entityType, entityID)
	if err != nil {
		return fmt.Errorf("%w: clear pending: %v", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ClearAllPending(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sync_pending`)
	if err != nil {
		return fmt.Errorf("%w: clear all pending: %v", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) MigrateUserID(ctx context.Context, from, to string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE movies SET user_id = ? WHERE user_id = ?`, to, from)
	if err != nil {
		return fmt.Errorf("%w: migrate user id: %v", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ApplyMovieLWW(ctx context.Context, movie domain.Movie) (bool, error) {
	_, applied, err := r.movies.SyncUpsert(ctx, movie)
	return applied, err
}

func (r *SyncRepository) ApplyWatchEntryLWW(ctx context.Context, entry domain.WatchEntry) (bool, error) {
	_, applied, err := r.entries.SyncUpsert(ctx, entry)
	return applied, err
}

func (r *SyncRepository) HasPendingDelete(ctx context.Context, movieID string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sync_pending
		WHERE entity_type = ? AND entity_id = ? AND operation = ?
	`, PendingEntityDelete, movieID, PendingOpDelete)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("%w: has pending delete: %v", apperrors.ErrDB, err)
	}
	return count > 0, nil
}
