package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestWatchEntryRepositorySyncUpsertLWW(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	entries := NewWatchEntryRepository(db)

	movie, err := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "Heat", Year: 1995})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	older := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	rating := 8.0

	first, applied, err := entries.SyncUpsert(ctx, domain.WatchEntry{
		MovieID:     movie.ID,
		Watched:     true,
		Rating:      &rating,
		RatingScale: 10,
		UpdatedAt:   older,
	})
	if err != nil || !applied || !first.Watched {
		t.Fatalf("first sync upsert: entry=%+v applied=%v err=%v", first, applied, err)
	}

	staleRating := 5.0
	stale, applied, err := entries.SyncUpsert(ctx, domain.WatchEntry{
		MovieID:     movie.ID,
		Watched:     true,
		Rating:      &staleRating,
		RatingScale: 10,
		UpdatedAt:   older.Add(-time.Minute),
	})
	if err != nil || applied {
		t.Fatalf("stale sync upsert should not apply: entry=%+v applied=%v err=%v", stale, applied, err)
	}
	if stale.Rating == nil || *stale.Rating != 8.0 {
		t.Fatalf("expected older rating kept, got %+v", stale.Rating)
	}

	newerRating := 9.0
	updated, applied, err := entries.SyncUpsert(ctx, domain.WatchEntry{
		MovieID:     movie.ID,
		Watched:     true,
		Rating:      &newerRating,
		RatingScale: 10,
		UpdatedAt:   newer,
	})
	if err != nil || !applied {
		t.Fatalf("newer sync upsert: entry=%+v applied=%v err=%v", updated, applied, err)
	}
	if updated.Rating == nil || *updated.Rating != 9.0 {
		t.Fatalf("expected newer rating, got %+v", updated.Rating)
	}
}

func TestWatchEntryRepositoryListByMovieIDs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	entries := NewWatchEntryRepository(db)

	m1, _ := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "A", Year: 2000})
	m2, _ := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "B", Year: 2001})

	if _, err := entries.Upsert(ctx, domain.WatchEntry{MovieID: m1.ID, Watched: true, RatingScale: 10}); err != nil {
		t.Fatalf("upsert entry 1: %v", err)
	}
	if _, err := entries.Upsert(ctx, domain.WatchEntry{MovieID: m2.ID, Watched: false, RatingScale: 10}); err != nil {
		t.Fatalf("upsert entry 2: %v", err)
	}

	list, err := entries.ListByMovieIDs(ctx, []string{m1.ID, m2.ID, "missing"})
	if err != nil {
		t.Fatalf("list by movie ids: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
}

func TestWatchEntryRepositoryDeleteByMovieID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	entries := NewWatchEntryRepository(db)

	movie, _ := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "Delete Me", Year: 1999})
	if _, err := entries.Upsert(ctx, domain.WatchEntry{MovieID: movie.ID, Watched: true, RatingScale: 10}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := entries.DeleteByMovieID(ctx, movie.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := entries.GetByMovieID(ctx, movie.ID)
	if err == nil || !errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}
