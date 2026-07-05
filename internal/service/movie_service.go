package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type MovieStore interface {
	Create(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	GetByID(ctx context.Context, id string) (domain.Movie, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Movie, error)
	Update(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	Delete(ctx context.Context, id string) error
}

type WatchEntryStore interface {
	Upsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error)
	GetByMovieID(ctx context.Context, movieID string) (domain.WatchEntry, error)
	DeleteByMovieID(ctx context.Context, movieID string) error
}

type MovieService struct {
	movies       MovieStore
	watchEntries WatchEntryStore
}

func NewMovieService(movies MovieStore, watchEntries WatchEntryStore) *MovieService {
	return &MovieService{
		movies:       movies,
		watchEntries: watchEntries,
	}
}

func (s *MovieService) CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	movie.Title = strings.TrimSpace(movie.Title)
	if err := validateMovie(movie); err != nil {
		return domain.Movie{}, err
	}

	return s.movies.Create(ctx, movie)
}

func (s *MovieService) GetMovie(ctx context.Context, id string) (domain.Movie, error) {
	return s.movies.GetByID(ctx, id)
}

func (s *MovieService) ListMovies(ctx context.Context, userID string) ([]domain.Movie, error) {
	return s.movies.ListByUser(ctx, userID)
}

func (s *MovieService) UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	movie.Title = strings.TrimSpace(movie.Title)
	if err := validateMovie(movie); err != nil {
		return domain.Movie{}, err
	}

	return s.movies.Update(ctx, movie)
}

func (s *MovieService) DeleteMovie(ctx context.Context, id string) error {
	return s.movies.Delete(ctx, id)
}

func (s *MovieService) SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	if entry.RatingScale == 0 {
		entry.RatingScale = 10
	}
	if err := validateWatchEntry(entry); err != nil {
		return domain.WatchEntry{}, err
	}

	return s.watchEntries.Upsert(ctx, entry)
}

func (s *MovieService) GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	return s.watchEntries.GetByMovieID(ctx, movieID)
}

func validateMovie(movie domain.Movie) error {
	if movie.UserID == "" {
		return fmt.Errorf("%w: user id is required", apperrors.ErrValidation)
	}
	if movie.Title == "" {
		return fmt.Errorf("%w: title is required", apperrors.ErrValidation)
	}
	if movie.Year < 0 {
		return fmt.Errorf("%w: year cannot be negative", apperrors.ErrValidation)
	}
	return nil
}

func validateWatchEntry(entry domain.WatchEntry) error {
	if entry.MovieID == "" {
		return fmt.Errorf("%w: movie id is required", apperrors.ErrValidation)
	}
	if entry.RatingScale <= 0 {
		return fmt.Errorf("%w: rating scale must be positive", apperrors.ErrInvalidRating)
	}
	if entry.Rating != nil && (*entry.Rating < 0 || *entry.Rating > float64(entry.RatingScale)) {
		return fmt.Errorf("%w: rating must be between 0 and %d", apperrors.ErrInvalidRating, entry.RatingScale)
	}
	return nil
}
