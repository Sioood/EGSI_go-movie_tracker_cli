package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type WatchEntryRepository struct {
	db *sql.DB
}

func NewWatchEntryRepository(db *sql.DB) *WatchEntryRepository {
	return &WatchEntryRepository{db: db}
}

func (r *WatchEntryRepository) Upsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.RatingScale == 0 {
		entry.RatingScale = 10
	}
	entry.UpdatedAt = now

	saved, _, err := r.insertWatchEntry(ctx, entry)
	return saved, err
}

// SyncUpsert applies a watch entry when incoming.UpdatedAt is newer than the existing row.
func (r *WatchEntryRepository) SyncUpsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	if entry.RatingScale == 0 {
		entry.RatingScale = 10
	}

	existing, err := r.GetByMovieID(ctx, entry.MovieID)
	if errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		if entry.ID == "" {
			entry.ID = uuid.NewString()
		}
		if entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = time.Now().UTC()
		}
		return r.insertWatchEntry(ctx, entry)
	}
	if err != nil {
		return domain.WatchEntry{}, false, err
	}

	if !entry.UpdatedAt.After(existing.UpdatedAt) {
		return existing, false, nil
	}

	if entry.ID == "" {
		entry.ID = existing.ID
	}
	return r.insertWatchEntry(ctx, entry)
}

func (r *WatchEntryRepository) insertWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	var watchedAt any
	if entry.WatchedAt != nil {
		watchedAt = formatTime(*entry.WatchedAt)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO watch_entries (id, movie_id, watched, rating, rating_scale, review, watched_at, updated_by_device, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(movie_id) DO UPDATE SET
			watched = excluded.watched,
			rating = excluded.rating,
			rating_scale = excluded.rating_scale,
			review = excluded.review,
			watched_at = excluded.watched_at,
			updated_by_device = excluded.updated_by_device,
			updated_at = excluded.updated_at
	`, entry.ID, entry.MovieID, entry.Watched, entry.Rating, entry.RatingScale, entry.Review, watchedAt, entry.UpdatedByDevice, formatTime(entry.UpdatedAt))
	if err != nil {
		return domain.WatchEntry{}, false, fmt.Errorf("%w: sync upsert watch entry: %v", apperrors.ErrDB, err)
	}

	saved, err := r.GetByMovieID(ctx, entry.MovieID)
	if err != nil {
		return domain.WatchEntry{}, false, err
	}
	return saved, true, nil
}

func (r *WatchEntryRepository) ListAll(ctx context.Context) ([]domain.WatchEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, movie_id, watched, rating, rating_scale, review, watched_at, updated_by_device, updated_at
		FROM watch_entries
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: list watch entries: %v", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var entries []domain.WatchEntry
	for rows.Next() {
		entry, err := scanWatchEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scan watch entry: %v", apperrors.ErrDB, err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate watch entries: %v", apperrors.ErrDB, err)
	}
	return entries, nil
}

func (r *WatchEntryRepository) GetByMovieID(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, movie_id, watched, rating, rating_scale, review, watched_at, updated_by_device, updated_at
		FROM watch_entries
		WHERE movie_id = ?
	`, movieID)

	entry, err := scanWatchEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WatchEntry{}, fmt.Errorf("%w: movie id %s", apperrors.ErrWatchEntryNotFound, movieID)
	}
	if err != nil {
		return domain.WatchEntry{}, fmt.Errorf("%w: get watch entry: %v", apperrors.ErrDB, err)
	}

	return entry, nil
}

func (r *WatchEntryRepository) DeleteByMovieID(ctx context.Context, movieID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM watch_entries WHERE movie_id = ?`, movieID)
	if err != nil {
		return fmt.Errorf("%w: delete watch entry: %v", apperrors.ErrDB, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: delete watch entry rows affected: %v", apperrors.ErrDB, err)
	}
	if count == 0 {
		return fmt.Errorf("%w: movie id %s", apperrors.ErrWatchEntryNotFound, movieID)
	}

	return nil
}

type watchEntryScanner interface {
	Scan(dest ...any) error
}

func scanWatchEntry(scanner watchEntryScanner) (domain.WatchEntry, error) {
	var entry domain.WatchEntry
	var watched int
	var rating sql.NullFloat64
	var watchedAt sql.NullString
	var updatedAt string

	if err := scanner.Scan(
		&entry.ID,
		&entry.MovieID,
		&watched,
		&rating,
		&entry.RatingScale,
		&entry.Review,
		&watchedAt,
		&entry.UpdatedByDevice,
		&updatedAt,
	); err != nil {
		return domain.WatchEntry{}, err
	}

	entry.Watched = watched != 0
	if rating.Valid {
		entry.Rating = &rating.Float64
	}

	parsedWatchedAt, err := parseNullableTime(watchedAt)
	if err != nil {
		return domain.WatchEntry{}, err
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return domain.WatchEntry{}, err
	}

	entry.WatchedAt = parsedWatchedAt
	entry.UpdatedAt = parsedUpdatedAt
	return entry, nil
}
