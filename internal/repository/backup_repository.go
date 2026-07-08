package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// UserBackup stores per-user JSON snapshots for config and UI state.
type UserBackup struct {
	UserID    string
	Config    string
	State     string
	UpdatedAt time.Time
}

type BackupRepository struct {
	db *sql.DB
}

func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

func (r *BackupRepository) Get(ctx context.Context, userID string) (UserBackup, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, config, state, updated_at
		FROM user_backups
		WHERE user_id = ?
	`, userID)

	var backup UserBackup
	var updatedAt string
	if err := row.Scan(&backup.UserID, &backup.Config, &backup.State, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserBackup{
				UserID:    userID,
				Config:    "{}",
				State:     "{}",
				UpdatedAt: time.Time{},
			}, nil
		}
		return UserBackup{}, fmt.Errorf("get backup: %w", err)
	}
	updated, err := parseTime(updatedAt)
	if err != nil {
		return UserBackup{}, err
	}
	backup.UpdatedAt = updated
	return backup, nil
}

func (r *BackupRepository) UpsertConfig(ctx context.Context, userID, configJSON string) error {
	now := formatTime(time.Now().UTC())
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_backups (user_id, config, state, updated_at)
		VALUES (?, ?, '{}', ?)
		ON CONFLICT(user_id) DO UPDATE SET
			config = excluded.config,
			updated_at = excluded.updated_at
	`, userID, configJSON, now)
	if err != nil {
		return fmt.Errorf("upsert backup config: %w", err)
	}
	return nil
}

func (r *BackupRepository) UpsertState(ctx context.Context, userID, stateJSON string) error {
	now := formatTime(time.Now().UTC())
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_backups (user_id, config, state, updated_at)
		VALUES (?, '{}', ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			state = excluded.state,
			updated_at = excluded.updated_at
	`, userID, stateJSON, now)
	if err != nil {
		return fmt.Errorf("upsert backup state: %w", err)
	}
	return nil
}

func (r *BackupRepository) UpsertFull(ctx context.Context, userID, configJSON, stateJSON string) error {
	now := formatTime(time.Now().UTC())
	if userID == "" {
		return fmt.Errorf("user id required")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_backups (user_id, config, state, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			config = excluded.config,
			state = excluded.state,
			updated_at = excluded.updated_at
	`, userID, configJSON, stateJSON, now)
	if err != nil {
		return fmt.Errorf("upsert full backup: %w", err)
	}
	return nil
}
