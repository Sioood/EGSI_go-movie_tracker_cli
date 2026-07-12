package service

import (
	"context"
	"errors"
	"strings"
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

	longReview := strings.Repeat("a", MaxReviewLength+1)
	_, err = service.SaveWatchEntry(context.Background(), domain.WatchEntry{
		MovieID:     "movie-1",
		RatingScale: 10,
		Review:      longReview,
	})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("expected review length validation error, got %v", err)
	}
}

func TestSyncUpsertMovieRejectsForeignOwner(t *testing.T) {
	store := syncTestMovieStore{
		existing: domain.Movie{ID: "movie-1", UserID: "other-user", Title: "Arrival"},
	}
	service := NewMovieService(store, fakeWatchEntryStore{})

	_, _, err := service.SyncUpsertMovie(context.Background(), "local-user", domain.Movie{
		ID:     "movie-1",
		UserID: "local-user",
		Title:  "Arrival",
		Year:   2016,
	})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestSyncUpsertMovieAppliesWhenOwned(t *testing.T) {
	store := syncTestMovieStore{
		existing: domain.Movie{ID: "movie-1", UserID: "local-user", Title: "Arrival"},
	}
	service := NewMovieService(store, fakeWatchEntryStore{})

	saved, applied, err := service.SyncUpsertMovie(context.Background(), "local-user", domain.Movie{
		ID:     "movie-1",
		UserID: "local-user",
		Title:  "Arrival Updated",
		Year:   2016,
	})
	if err != nil || !applied || saved.Title != "Arrival Updated" {
		t.Fatalf("sync upsert movie: saved=%+v applied=%v err=%v", saved, applied, err)
	}
}

func TestSyncUpsertWatchEntryValidatesReviewLength(t *testing.T) {
	service := NewMovieService(fakeMovieStore{}, fakeWatchEntryStore{})
	longReview := strings.Repeat("x", MaxReviewLength+1)

	_, _, err := service.SyncUpsertWatchEntry(context.Background(), domain.WatchEntry{
		MovieID:     "movie-1",
		RatingScale: 10,
		Review:      longReview,
	})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

type syncTestMovieStore struct {
	existing domain.Movie
	fakeMovieStore
}

func (s syncTestMovieStore) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	if s.existing.ID == id {
		return s.existing, nil
	}
	return domain.Movie{}, apperrors.ErrMovieNotFound
}

type fakeMovieStore struct{}

func (fakeMovieStore) Create(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	return movie, nil
}

func (fakeMovieStore) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	return domain.Movie{ID: id}, nil
}

func (fakeMovieStore) GetByExternalID(ctx context.Context, userID, externalID string) (domain.Movie, error) {
	return domain.Movie{}, apperrors.ErrMovieNotFound
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

func (fakeWatchEntryStore) ListByMovieIDs(ctx context.Context, movieIDs []string) ([]domain.WatchEntry, error) {
	return nil, nil
}

func (fakeWatchEntryStore) DeleteByMovieID(ctx context.Context, movieID string) error {
	return nil
}

func (fakeWatchEntryStore) SyncUpsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	return entry, true, nil
}
