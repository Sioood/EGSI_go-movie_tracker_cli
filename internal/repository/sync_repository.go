package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	LastSyncAt     *time.Time
	LastPushAt     *time.Time
	LastPullAt     *time.Time
	UserIDMigrated bool
}

// PendingChange is a local mutation waiting to be pushed.
type PendingChange struct {
	EntityType string
	EntityID   string
	Operation  string
}

type SyncRepository struct {
	db      *sql.DB
	movies  *MovieRepository
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
		return SyncMetadata{}, fmt.Errorf("%w: get sync metadata: %w", apperrors.ErrDB, err)
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
		return fmt.Errorf("%w: update sync metadata: %w", apperrors.ErrDB, err)
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
		return fmt.Errorf("%w: mark pending: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ListPending(ctx context.Context) ([]PendingChange, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT entity_type, entity_id, operation
		FROM sync_pending
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: list pending: %w", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var pending []PendingChange
	for rows.Next() {
		var item PendingChange
		if err := rows.Scan(&item.EntityType, &item.EntityID, &item.Operation); err != nil {
			return nil, fmt.Errorf("%w: scan pending: %w", apperrors.ErrDB, err)
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate pending: %w", apperrors.ErrDB, err)
	}
	return pending, nil
}

func (r *SyncRepository) PendingCount(ctx context.Context) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_pending`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: pending count: %w", apperrors.ErrDB, err)
	}
	return count, nil
}

func (r *SyncRepository) ClearPending(ctx context.Context, entityType, entityID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM sync_pending WHERE entity_type = ? AND entity_id = ?
	`, entityType, entityID)
	if err != nil {
		return fmt.Errorf("%w: clear pending: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ClearAllPending(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sync_pending`)
	if err != nil {
		return fmt.Errorf("%w: clear all pending: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) MigrateUserID(ctx context.Context, from, to string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE movies SET user_id = ? WHERE user_id = ?`, to, from)
	if err != nil {
		return fmt.Errorf("%w: migrate user id: %w", apperrors.ErrDB, err)
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
		return false, fmt.Errorf("%w: has pending delete: %w", apperrors.ErrDB, err)
	}
	return count > 0, nil
}

func (r *SyncRepository) HasPending(ctx context.Context, entityType, entityID string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sync_pending
		WHERE entity_type = ? AND entity_id = ?
	`, entityType, entityID)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("%w: has pending: %w", apperrors.ErrDB, err)
	}
	return count > 0, nil
}

func (r *SyncRepository) HasOpenConflict(ctx context.Context, entityType, entityID string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sync_conflicts
		WHERE entity_type = ? AND entity_id = ? AND resolved_at IS NULL
	`, entityType, entityID)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("%w: has open conflict: %w", apperrors.ErrDB, err)
	}
	return count > 0, nil
}

func (r *SyncRepository) RecordConflict(ctx context.Context, conflict domain.SyncConflict) error {
	if conflict.ID == "" {
		conflict.ID = uuid.NewString()
	}
	if conflict.DetectedAt.IsZero() {
		conflict.DetectedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_conflicts (
			id, entity_type, entity_id, local_json, remote_json,
			local_device_id, remote_device_id, detected_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(id) DO NOTHING
	`, conflict.ID, conflict.EntityType, conflict.EntityID, conflict.LocalJSON, conflict.RemoteJSON,
		conflict.LocalDeviceID, conflict.RemoteDeviceID, formatTime(conflict.DetectedAt))
	if err != nil {
		return fmt.Errorf("%w: record conflict: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) ConflictCount(ctx context.Context) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_conflicts WHERE resolved_at IS NULL`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: conflict count: %w", apperrors.ErrDB, err)
	}
	return count, nil
}

func (r *SyncRepository) ListConflicts(ctx context.Context) ([]domain.SyncConflict, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, entity_type, entity_id, local_json, remote_json,
			local_device_id, remote_device_id, detected_at, resolved_at
		FROM sync_conflicts
		WHERE resolved_at IS NULL
		ORDER BY detected_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: list conflicts: %w", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var conflicts []domain.SyncConflict
	for rows.Next() {
		var conflict domain.SyncConflict
		var detectedAt string
		var resolvedAt sql.NullString
		if err := rows.Scan(
			&conflict.ID,
			&conflict.EntityType,
			&conflict.EntityID,
			&conflict.LocalJSON,
			&conflict.RemoteJSON,
			&conflict.LocalDeviceID,
			&conflict.RemoteDeviceID,
			&detectedAt,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: scan conflict: %w", apperrors.ErrDB, err)
		}
		parsedDetected, err := parseTime(detectedAt)
		if err != nil {
			return nil, err
		}
		conflict.DetectedAt = parsedDetected
		if resolvedAt.Valid {
			parsedResolved, err := parseTime(resolvedAt.String)
			if err != nil {
				return nil, err
			}
			conflict.ResolvedAt = &parsedResolved
		}
		conflicts = append(conflicts, conflict)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate conflicts: %w", apperrors.ErrDB, err)
	}
	return conflicts, nil
}

func (r *SyncRepository) ResolveConflict(ctx context.Context, id, choice string) error {
	row := r.db.QueryRowContext(ctx, `
		SELECT entity_type, entity_id, local_json, remote_json
		FROM sync_conflicts
		WHERE id = ? AND resolved_at IS NULL
	`, id)

	var entityType, entityID, localJSON, remoteJSON string
	if err := row.Scan(&entityType, &entityID, &localJSON, &remoteJSON); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: conflict not found", apperrors.ErrValidation)
		}
		return fmt.Errorf("%w: get conflict: %w", apperrors.ErrDB, err)
	}

	switch entityType {
	case domain.SyncEntityMovie:
		var chosen domain.Movie
		source := remoteJSON
		if choice == domain.ConflictChoiceLocal {
			source = localJSON
		}
		if err := json.Unmarshal([]byte(source), &chosen); err != nil {
			return fmt.Errorf("parse conflict movie: %w", err)
		}
		if _, err := r.ApplyMovieLWW(ctx, chosen); err != nil {
			return err
		}
		if err := r.MarkPending(ctx, PendingEntityMovie, chosen.ID, PendingOpUpsert); err != nil {
			return fmt.Errorf("mark pending movie after conflict resolution: %w", err)
		}
	case domain.SyncEntityWatchEntry:
		var chosen domain.WatchEntry
		source := remoteJSON
		if choice == domain.ConflictChoiceLocal {
			source = localJSON
		}
		if err := json.Unmarshal([]byte(source), &chosen); err != nil {
			return fmt.Errorf("parse conflict watch entry: %w", err)
		}
		if _, err := r.ApplyWatchEntryLWW(ctx, chosen); err != nil {
			return err
		}
		if err := r.MarkPending(ctx, PendingEntityWatchEntry, chosen.MovieID, PendingOpUpsert); err != nil {
			return fmt.Errorf("mark pending watch entry after conflict resolution: %w", err)
		}
	default:
		return fmt.Errorf("%w: unknown conflict entity %s", apperrors.ErrValidation, entityType)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_conflicts SET resolved_at = ? WHERE id = ?
	`, formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("%w: resolve conflict: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) UpsertDevice(ctx context.Context, deviceID, deviceName string) error {
	if deviceID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_devices (device_id, device_name, last_seen_at)
		VALUES (?, ?, ?)
		ON CONFLICT(device_id) DO UPDATE SET
			device_name = CASE WHEN excluded.device_name != '' THEN excluded.device_name ELSE sync_devices.device_name END,
			last_seen_at = excluded.last_seen_at
	`, deviceID, deviceName, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("%w: upsert device: %w", apperrors.ErrDB, err)
	}
	return nil
}

func (r *SyncRepository) GetDeviceName(ctx context.Context, deviceID string) (string, error) {
	if deviceID == "" {
		return "", nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT device_name FROM sync_devices WHERE device_id = ?`, deviceID)
	var name string
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return deviceID, nil
		}
		return "", fmt.Errorf("%w: get device name: %w", apperrors.ErrDB, err)
	}
	if name == "" {
		return deviceID, nil
	}
	return name, nil
}
