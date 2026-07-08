package service

import (
	"context"
	"errors"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestMovieServiceValidation(t *testing.T) {
	service := NewMovieService(fakeMovieStore{}, fakeWatchEntryStore{})

	_, err := service.CreateMovie(context.Background(), domain.Movie{UserID: "user-1", Title: "   "})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("expected title validation error, got %v", err)
	}

	rating := 11.0
	_, err = service.SaveWatchEntry(context.Background(), domain.WatchEntry{
		MovieID:     "movie-1",
		Rating:      &rating,
		RatingScale: 10,
	})
	if !errors.Is(err, apperrors.ErrInvalidRating) {
		t.Fatalf("expected invalid rating error, got %v", err)
	}
}

type fakeMovieStore struct{}

func (fakeMovieStore) Create(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	return movie, nil
}

func (fakeMovieStore) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	return domain.Movie{ID: id}, nil
}

func (fakeMovieStore) ListByUser(ctx context.Context, userID string) ([]domain.Movie, error) {
	return []domain.Movie{{UserID: userID}}, nil
}

func (fakeMovieStore) Search(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	return []domain.Movie{{UserID: params.UserID, Title: params.Query}}, nil
}

func (fakeMovieStore) Update(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	return movie, nil
}

func (fakeMovieStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (fakeMovieStore) SyncUpsert(ctx context.Context, movie domain.Movie) (domain.Movie, bool, error) {
	return movie, true, nil
}

type fakeWatchEntryStore struct{}

func (fakeWatchEntryStore) Upsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	return entry, nil
}

func (fakeWatchEntryStore) GetByMovieID(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	return domain.WatchEntry{MovieID: movieID}, nil
}

func (fakeWatchEntryStore) DeleteByMovieID(ctx context.Context, movieID string) error {
	return nil
}

func (fakeWatchEntryStore) SyncUpsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	return entry, true, nil
}
