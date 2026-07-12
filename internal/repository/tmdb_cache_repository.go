package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/tmdb"
)

const tmdbCacheTTL = 7 * 24 * time.Hour

// TMDBCacheRepository stores TMDB metadata locally.
type TMDBCacheRepository struct {
	db *sql.DB
}

// NewTMDBCacheRepository creates a TMDB cache repository.
func NewTMDBCacheRepository(db *sql.DB) *TMDBCacheRepository {
	return &TMDBCacheRepository{db: db}
}

// Get returns a cached search result when still fresh.
func (r *TMDBCacheRepository) Get(ctx context.Context, tmdbID int) (tmdb.SearchResult, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT payload_json, fetched_at
		FROM tmdb_cache
		WHERE tmdb_id = ?
	`, tmdbID)

	var payloadJSON string
	var fetchedAt string
	if err := row.Scan(&payloadJSON, &fetchedAt); err != nil {
		if err == sql.ErrNoRows {
			return tmdb.SearchResult{}, false, nil
		}
		return tmdb.SearchResult{}, false, fmt.Errorf("%w: get tmdb cache: %w", apperrors.ErrDB, err)
	}

	fetched, err := parseTime(fetchedAt)
	if err != nil {
		return tmdb.SearchResult{}, false, err
	}
	if time.Since(fetched) > tmdbCacheTTL {
		return tmdb.SearchResult{}, false, nil
	}

	var payload tmdb.CachedPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return tmdb.SearchResult{}, false, fmt.Errorf("parse tmdb cache: %w", err)
	}
	return payload.Result, true, nil
}

// Put stores a TMDB search result in the cache.
func (r *TMDBCacheRepository) Put(ctx context.Context, result tmdb.SearchResult) error {
	payload := tmdb.CachedPayload{
		Result:    result,
		FetchedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tmdb cache: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tmdb_cache (tmdb_id, payload_json, fetched_at)
		VALUES (?, ?, ?)
		ON CONFLICT(tmdb_id) DO UPDATE SET
			payload_json = excluded.payload_json,
			fetched_at = excluded.fetched_at
	`, result.ID, string(data), formatTime(payload.FetchedAt))
	if err != nil {
		return fmt.Errorf("%w: put tmdb cache: %w", apperrors.ErrDB, err)
	}
	return nil
}

// CacheSearchResults stores each result from a search response.
func (r *TMDBCacheRepository) CacheSearchResults(ctx context.Context, results []tmdb.SearchResult) error {
	for _, result := range results {
		if err := r.Put(ctx, result); err != nil {
			return err
		}
	}
	return nil
}
