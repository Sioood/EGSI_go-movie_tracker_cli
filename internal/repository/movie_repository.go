package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type MovieRepository struct {
	db *sql.DB
}

func NewMovieRepository(db *sql.DB) *MovieRepository {
	return &MovieRepository{db: db}
}

func (r *MovieRepository) Create(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	now := time.Now().UTC()
	if movie.ID == "" {
		movie.ID = uuid.NewString()
	}
	if movie.CreatedAt.IsZero() {
		movie.CreatedAt = now
	}
	movie.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO movies (id, user_id, title, year, external_id, updated_by_device, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, movie.ID, movie.UserID, movie.Title, movie.Year, movie.ExternalID, movie.UpdatedByDevice, formatTime(movie.CreatedAt), formatTime(movie.UpdatedAt))
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: create movie: %w", apperrors.ErrDB, err)
	}

	return movie, nil
}

func (r *MovieRepository) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT movies.id, movies.user_id, movies.title, movies.year, movies.external_id, movies.updated_by_device, movies.created_at, movies.updated_at
		FROM movies
		WHERE id = ?
	`, id)

	movie, err := scanMovie(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Movie{}, fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, id)
	}
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: get movie: %w", apperrors.ErrDB, err)
	}

	return movie, nil
}

func (r *MovieRepository) GetByExternalID(ctx context.Context, userID, externalID string) (domain.Movie, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return domain.Movie{}, fmt.Errorf("%w: external id is required", apperrors.ErrValidation)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT movies.id, movies.user_id, movies.title, movies.year, movies.external_id, movies.updated_by_device, movies.created_at, movies.updated_at
		FROM movies
		WHERE movies.user_id = ? AND movies.external_id = ?
	`, userID, externalID)

	movie, err := scanMovie(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Movie{}, fmt.Errorf("%w: external id %s", apperrors.ErrMovieNotFound, externalID)
	}
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: get movie by external id: %w", apperrors.ErrDB, err)
	}
	return movie, nil
}

func (r *MovieRepository) ListByUser(ctx context.Context, userID string) ([]domain.Movie, error) {
	return r.Search(ctx, domain.MovieSearchParams{UserID: userID, Filter: domain.MovieFilterAll, Sort: domain.MovieSortTitle})
}

func (r *MovieRepository) Search(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	if params.Filter == "" {
		params.Filter = domain.MovieFilterAll
	}
	if params.Sort == "" {
		params.Sort = domain.MovieSortTitle
	}

	query := `
		SELECT movies.id, movies.user_id, movies.title, movies.year, movies.external_id, movies.updated_by_device, movies.created_at, movies.updated_at
		FROM movies
		LEFT JOIN watch_entries ON watch_entries.movie_id = movies.id
		WHERE movies.user_id = ?
	`
	args := []any{params.UserID}

	if strings.TrimSpace(params.Query) != "" {
		query += ` AND movies.title LIKE ? ESCAPE '\'`
		args = append(args, "%"+escapeLike(strings.TrimSpace(params.Query))+"%")
	}

	switch params.Filter {
	case domain.MovieFilterWatched:
		query += ` AND COALESCE(watch_entries.watched, 0) = 1`
	case domain.MovieFilterUnwatched:
		query += ` AND COALESCE(watch_entries.watched, 0) = 0`
	case domain.MovieFilterRated:
		query += ` AND watch_entries.rating IS NOT NULL`
	case domain.MovieFilterUnrated:
		query += ` AND watch_entries.rating IS NULL`
	case domain.MovieFilterAll:
	default:
		return nil, fmt.Errorf("%w: unknown movie filter %q", apperrors.ErrValidation, params.Filter)
	}

	switch params.Sort {
	case domain.MovieSortTitle:
		query += ` ORDER BY movies.title COLLATE NOCASE ASC, movies.created_at ASC`
	case domain.MovieSortDate:
		query += ` ORDER BY watch_entries.watched_at IS NULL ASC, watch_entries.watched_at DESC, movies.title COLLATE NOCASE ASC`
	case domain.MovieSortRating:
		query += ` ORDER BY watch_entries.rating IS NULL ASC, watch_entries.rating DESC, movies.title COLLATE NOCASE ASC`
	default:
		return nil, fmt.Errorf("%w: unknown movie sort %q", apperrors.ErrValidation, params.Sort)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: search movies: %w", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var movies []domain.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scan movie: %w", apperrors.ErrDB, err)
		}
		movies = append(movies, movie)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate movies: %w", apperrors.ErrDB, err)
	}

	return movies, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func (r *MovieRepository) Update(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	movie.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, `
		UPDATE movies
		SET user_id = ?, title = ?, year = ?, external_id = ?, updated_by_device = ?, updated_at = ?
		WHERE id = ?
	`, movie.UserID, movie.Title, movie.Year, movie.ExternalID, movie.UpdatedByDevice, formatTime(movie.UpdatedAt), movie.ID)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: update movie: %w", apperrors.ErrDB, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: update movie rows affected: %w", apperrors.ErrDB, err)
	}
	if count == 0 {
		return domain.Movie{}, fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, movie.ID)
	}

	return r.GetByID(ctx, movie.ID)
}

func (r *MovieRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM movies WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("%w: delete movie: %w", apperrors.ErrDB, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: delete movie rows affected: %w", apperrors.ErrDB, err)
	}
	if count == 0 {
		return fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, id)
	}

	return nil
}

// SyncUpsert applies a movie when incoming.UpdatedAt is newer than the existing row.
// On equal timestamps the existing row wins. Returns applied=false when skipped.
func (r *MovieRepository) SyncUpsert(ctx context.Context, movie domain.Movie) (domain.Movie, bool, error) {
	existing, err := r.GetByID(ctx, movie.ID)
	if errors.Is(err, apperrors.ErrMovieNotFound) {
		if movie.ID == "" {
			movie.ID = uuid.NewString()
		}
		now := time.Now().UTC()
		if movie.CreatedAt.IsZero() {
			movie.CreatedAt = now
		}
		if movie.UpdatedAt.IsZero() {
			movie.UpdatedAt = now
		}

		_, err := r.db.ExecContext(ctx, `
			INSERT INTO movies (id, user_id, title, year, external_id, updated_by_device, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, movie.ID, movie.UserID, movie.Title, movie.Year, movie.ExternalID, movie.UpdatedByDevice, formatTime(movie.CreatedAt), formatTime(movie.UpdatedAt))
		if err != nil {
			return domain.Movie{}, false, fmt.Errorf("%w: sync insert movie: %w", apperrors.ErrDB, err)
		}
		return movie, true, nil
	}
	if err != nil {
		return domain.Movie{}, false, err
	}

	if !movie.UpdatedAt.After(existing.UpdatedAt) {
		return existing, false, nil
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE movies
		SET user_id = ?, title = ?, year = ?, external_id = ?, updated_by_device = ?, updated_at = ?
		WHERE id = ?
	`, movie.UserID, movie.Title, movie.Year, movie.ExternalID, movie.UpdatedByDevice, formatTime(movie.UpdatedAt), movie.ID)
	if err != nil {
		return domain.Movie{}, false, fmt.Errorf("%w: sync update movie: %w", apperrors.ErrDB, err)
	}

	updated, err := r.GetByID(ctx, movie.ID)
	if err != nil {
		return domain.Movie{}, false, err
	}
	return updated, true, nil
}

type movieScanner interface {
	Scan(dest ...any) error
}

func scanMovie(scanner movieScanner) (domain.Movie, error) {
	var movie domain.Movie
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&movie.ID,
		&movie.UserID,
		&movie.Title,
		&movie.Year,
		&movie.ExternalID,
		&movie.UpdatedByDevice,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Movie{}, err
	}

	created, err := parseTime(createdAt)
	if err != nil {
		return domain.Movie{}, err
	}
	updated, err := parseTime(updatedAt)
	if err != nil {
		return domain.Movie{}, err
	}

	movie.CreatedAt = created
	movie.UpdatedAt = updated
	return movie, nil
}
