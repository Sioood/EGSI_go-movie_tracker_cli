package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

// MaxReviewLength is the maximum number of characters allowed in a watch entry review.
const MaxReviewLength = 5000

type MovieStore interface {
	Create(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	GetByID(ctx context.Context, id string) (domain.Movie, error)
	GetByExternalID(ctx context.Context, userID, externalID string) (domain.Movie, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Movie, error)
	Search(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error)
	Update(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	Delete(ctx context.Context, id string) error
	SyncUpsert(ctx context.Context, movie domain.Movie) (domain.Movie, bool, error)
}

type WatchEntryStore interface {
	Upsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error)
	GetByMovieID(ctx context.Context, movieID string) (domain.WatchEntry, error)
	ListByMovieIDs(ctx context.Context, movieIDs []string) ([]domain.WatchEntry, error)
	DeleteByMovieID(ctx context.Context, movieID string) error
	SyncUpsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error)
}

// MovieWithEntry pairs a movie with an optional watch entry.
type MovieWithEntry struct {
	Movie      domain.Movie
	WatchEntry *domain.WatchEntry
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

	if movie.ExternalID != "" {
		if _, err := s.movies.GetByExternalID(ctx, movie.UserID, movie.ExternalID); err == nil {
			return domain.Movie{}, fmt.Errorf("%w: ce film TMDB est déjà dans la collection", apperrors.ErrValidation)
		} else if !errors.Is(err, apperrors.ErrMovieNotFound) {
			return domain.Movie{}, err
		}
	}

	return s.movies.Create(ctx, movie)
}

func (s *MovieService) GetMovie(ctx context.Context, id string) (domain.Movie, error) {
	return s.movies.GetByID(ctx, id)
}

func (s *MovieService) GetMovieForUser(ctx context.Context, userID, movieID string) (domain.Movie, error) {
	movie, err := s.movies.GetByID(ctx, movieID)
	if err != nil {
		return domain.Movie{}, err
	}
	if movie.UserID != userID {
		return domain.Movie{}, apperrors.ErrForbidden
	}
	return movie, nil
}

func (s *MovieService) GetMovieWithEntry(ctx context.Context, userID, movieID string) (MovieWithEntry, error) {
	movie, err := s.GetMovieForUser(ctx, userID, movieID)
	if err != nil {
		return MovieWithEntry{}, err
	}
	item := MovieWithEntry{Movie: movie}
	entry, err := s.watchEntries.GetByMovieID(ctx, movieID)
	if err == nil {
		item.WatchEntry = &entry
	} else if !errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		return MovieWithEntry{}, err
	}
	return item, nil
}

func (s *MovieService) ListMoviesWithEntries(ctx context.Context, userID string) ([]MovieWithEntry, error) {
	movies, err := s.movies.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.attachEntriesForMovies(ctx, movies)
}

func (s *MovieService) SearchMoviesWithEntries(ctx context.Context, params domain.MovieSearchParams) ([]MovieWithEntry, error) {
	movies, err := s.movies.Search(ctx, params)
	if err != nil {
		return nil, err
	}
	return s.attachEntriesForMovies(ctx, movies)
}

func (s *MovieService) attachEntriesForMovies(ctx context.Context, movies []domain.Movie) ([]MovieWithEntry, error) {
	if len(movies) == 0 {
		return nil, nil
	}
	movieIDs := make([]string, len(movies))
	for i, movie := range movies {
		movieIDs[i] = movie.ID
	}
	entries, err := s.watchEntries.ListByMovieIDs(ctx, movieIDs)
	if err != nil {
		return nil, err
	}
	byMovieID := make(map[string]domain.WatchEntry, len(entries))
	for _, entry := range entries {
		byMovieID[entry.MovieID] = entry
	}
	result := make([]MovieWithEntry, 0, len(movies))
	for _, movie := range movies {
		item := MovieWithEntry{Movie: movie}
		if entry, ok := byMovieID[movie.ID]; ok {
			entry := entry
			item.WatchEntry = &entry
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *MovieService) ListMovies(ctx context.Context, userID string) ([]domain.Movie, error) {
	return s.movies.ListByUser(ctx, userID)
}

func (s *MovieService) SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	params.Query = strings.TrimSpace(params.Query)
	if params.UserID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrors.ErrValidation)
	}
	return s.movies.Search(ctx, params)
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

// SyncUpsertMovie imports a movie during sync with last-write-wins semantics.
func (s *MovieService) SyncUpsertMovie(ctx context.Context, ownerID string, movie domain.Movie) (domain.Movie, bool, error) {
	movie.Title = strings.TrimSpace(movie.Title)
	if err := validateMovie(movie); err != nil {
		return domain.Movie{}, false, err
	}

	existing, err := s.movies.GetByID(ctx, movie.ID)
	if err == nil && existing.UserID != ownerID {
		return domain.Movie{}, false, fmt.Errorf("%w: movie belongs to another user", apperrors.ErrForbidden)
	}

	movie.UserID = ownerID
	return s.movies.SyncUpsert(ctx, movie)
}

// SyncUpsertWatchEntry imports a watch entry during sync with last-write-wins semantics.
func (s *MovieService) SyncUpsertWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	if entry.RatingScale == 0 {
		entry.RatingScale = 10
	}
	if err := validateWatchEntry(entry); err != nil {
		return domain.WatchEntry{}, false, err
	}
	return s.watchEntries.SyncUpsert(ctx, entry)
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
	if len(entry.Review) > MaxReviewLength {
		return fmt.Errorf("%w: review must be at most %d characters", apperrors.ErrValidation, MaxReviewLength)
	}
	return nil
}
