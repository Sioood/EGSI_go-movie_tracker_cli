package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type StatsRepository struct {
	db *sql.DB
}

func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) GetStats(ctx context.Context, userID string) (domain.Stats, error) {
	var stats domain.Stats

	// Total movies tracked
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM movies WHERE user_id = ?`, userID,
	).Scan(&stats.TotalMovies); err != nil {
		return domain.Stats{}, fmt.Errorf("%w: count movies: %w", apperrors.ErrDB, err)
	}

	// Watched count, rated count, average rating
	var avgRating sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN w.watched = 1 THEN 1 END),
			COUNT(w.rating),
			AVG(w.rating)
		FROM movies m
		LEFT JOIN watch_entries w ON w.movie_id = m.id
		WHERE m.user_id = ?
	`, userID).Scan(&stats.TotalWatched, &stats.TotalRated, &avgRating); err != nil {
		return domain.Stats{}, fmt.Errorf("%w: aggregate stats: %w", apperrors.ErrDB, err)
	}
	if avgRating.Valid {
		stats.AverageRating = avgRating.Float64
	}

	// Best movies (top 3 by rating)
	bestRows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.user_id, m.title, m.year, m.external_id, m.updated_by_device, m.created_at, m.updated_at, w.rating
		FROM movies m
		JOIN watch_entries w ON w.movie_id = m.id
		WHERE m.user_id = ? AND w.rating IS NOT NULL
		ORDER BY w.rating DESC, m.title COLLATE NOCASE ASC
		LIMIT 3
	`, userID)
	if err != nil {
		return domain.Stats{}, fmt.Errorf("%w: best movies: %w", apperrors.ErrDB, err)
	}
	defer bestRows.Close()
	stats.BestMovies, err = scanMovieRatings(bestRows)
	if err != nil {
		return domain.Stats{}, err
	}

	// Worst movies (bottom 3 by rating)
	worstRows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.user_id, m.title, m.year, m.external_id, m.updated_by_device, m.created_at, m.updated_at, w.rating
		FROM movies m
		JOIN watch_entries w ON w.movie_id = m.id
		WHERE m.user_id = ? AND w.rating IS NOT NULL
		ORDER BY w.rating ASC, m.title COLLATE NOCASE ASC
		LIMIT 3
	`, userID)
	if err != nil {
		return domain.Stats{}, fmt.Errorf("%w: worst movies: %w", apperrors.ErrDB, err)
	}
	defer worstRows.Close()
	stats.WorstMovies, err = scanMovieRatings(worstRows)
	if err != nil {
		return domain.Stats{}, err
	}

	// Monthly histogram of watched movies
	histRows, err := r.db.QueryContext(ctx, `
		SELECT strftime('%Y', w.watched_at) AS yr,
		       strftime('%m', w.watched_at) AS mo,
		       COUNT(*) AS cnt
		FROM movies m
		JOIN watch_entries w ON w.movie_id = m.id
		WHERE m.user_id = ? AND w.watched = 1 AND w.watched_at IS NOT NULL
		GROUP BY yr, mo
		ORDER BY yr ASC, mo ASC
	`, userID)
	if err != nil {
		return domain.Stats{}, fmt.Errorf("%w: monthly histogram: %w", apperrors.ErrDB, err)
	}
	defer histRows.Close()

	for histRows.Next() {
		var yearStr, monthStr string
		var count int
		if err := histRows.Scan(&yearStr, &monthStr, &count); err != nil {
			return domain.Stats{}, fmt.Errorf("%w: scan histogram row: %w", apperrors.ErrDB, err)
		}
		t, err := time.Parse("2006-01", yearStr+"-"+monthStr)
		if err != nil {
			continue
		}
		stats.ByMonth = append(stats.ByMonth, domain.MonthBucket{
			Year:  t.Year(),
			Month: t.Month(),
			Count: count,
		})
	}
	if err := histRows.Err(); err != nil {
		return domain.Stats{}, fmt.Errorf("%w: iterate histogram: %w", apperrors.ErrDB, err)
	}

	return stats, nil
}

func scanMovieRatings(rows *sql.Rows) ([]domain.MovieRating, error) {
	var result []domain.MovieRating
	for rows.Next() {
		var mr domain.MovieRating
		var createdAt, updatedAt string
		if err := rows.Scan(
			&mr.Movie.ID, &mr.Movie.UserID, &mr.Movie.Title, &mr.Movie.Year,
			&mr.Movie.ExternalID, &mr.Movie.UpdatedByDevice, &createdAt, &updatedAt, &mr.Rating,
		); err != nil {
			return nil, fmt.Errorf("%w: scan movie rating: %w", apperrors.ErrDB, err)
		}
		ct, err := parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		ut, err := parseTime(updatedAt)
		if err != nil {
			return nil, err
		}
		mr.Movie.CreatedAt = ct
		mr.Movie.UpdatedAt = ut
		result = append(result, mr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate movie ratings: %w", apperrors.ErrDB, err)
	}
	return result, nil
}
