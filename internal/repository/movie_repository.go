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
		INSERT INTO movies (id, user_id, title, year, external_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, movie.ID, movie.UserID, movie.Title, movie.Year, movie.ExternalID, formatTime(movie.CreatedAt), formatTime(movie.UpdatedAt))
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: create movie: %v", apperrors.ErrDB, err)
	}

	return movie, nil
}

func (r *MovieRepository) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, year, external_id, created_at, updated_at
		FROM movies
		WHERE id = ?
	`, id)

	movie, err := scanMovie(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Movie{}, fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, id)
	}
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: get movie: %v", apperrors.ErrDB, err)
	}

	return movie, nil
}

func (r *MovieRepository) ListByUser(ctx context.Context, userID string) ([]domain.Movie, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, title, year, external_id, created_at, updated_at
		FROM movies
		WHERE user_id = ?
		ORDER BY title COLLATE NOCASE ASC, created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: list movies: %v", apperrors.ErrDB, err)
	}
	defer rows.Close()

	var movies []domain.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scan movie: %v", apperrors.ErrDB, err)
		}
		movies = append(movies, movie)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate movies: %v", apperrors.ErrDB, err)
	}

	return movies, nil
}

func (r *MovieRepository) Update(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	movie.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, `
		UPDATE movies
		SET user_id = ?, title = ?, year = ?, external_id = ?, updated_at = ?
		WHERE id = ?
	`, movie.UserID, movie.Title, movie.Year, movie.ExternalID, formatTime(movie.UpdatedAt), movie.ID)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: update movie: %v", apperrors.ErrDB, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return domain.Movie{}, fmt.Errorf("%w: update movie rows affected: %v", apperrors.ErrDB, err)
	}
	if count == 0 {
		return domain.Movie{}, fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, movie.ID)
	}

	return r.GetByID(ctx, movie.ID)
}

func (r *MovieRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM movies WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("%w: delete movie: %v", apperrors.ErrDB, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: delete movie rows affected: %v", apperrors.ErrDB, err)
	}
	if count == 0 {
		return fmt.Errorf("%w: id %s", apperrors.ErrMovieNotFound, id)
	}

	return nil
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
