package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestMovieRepositoryCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	watchEntries := NewWatchEntryRepository(db)

	created, err := movies.Create(ctx, domain.Movie{
		UserID:     "user-1",
		Title:      "Arrival",
		Year:       2016,
		ExternalID: "tmdb:329865",
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated movie id")
	}

	got, err := movies.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get movie: %v", err)
	}
	if got.Title != "Arrival" || got.UserID != "user-1" || got.Year != 2016 {
		t.Fatalf("unexpected movie: %+v", got)
	}

	_, err = movies.Create(ctx, domain.Movie{UserID: "user-2", Title: "Heat", Year: 1995})
	if err != nil {
		t.Fatalf("create second user movie: %v", err)
	}

	list, err := movies.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list movies: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected one movie for user-1, got %+v", list)
	}

	created.Title = "Arrival Updated"
	created.Year = 2017
	updated, err := movies.Update(ctx, created)
	if err != nil {
		t.Fatalf("update movie: %v", err)
	}
	if updated.Title != "Arrival Updated" || updated.Year != 2017 {
		t.Fatalf("unexpected updated movie: %+v", updated)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) && !updated.UpdatedAt.Equal(updated.CreatedAt) {
		t.Fatalf("updated_at should not be before created_at: %+v", updated)
	}

	rating := 8.5
	watchedAt := time.Date(2026, 7, 4, 20, 30, 0, 0, time.UTC)
	entry, err := watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID:     created.ID,
		Watched:     true,
		Rating:      &rating,
		RatingScale: 10,
		Review:      "Smart and quiet sci-fi.",
		WatchedAt:   &watchedAt,
	})
	if err != nil {
		t.Fatalf("upsert watch entry: %v", err)
	}
	if entry.ID == "" || !entry.Watched || entry.Rating == nil || *entry.Rating != rating {
		t.Fatalf("unexpected watch entry: %+v", entry)
	}

	rating = 9
	entry.Rating = &rating
	entry.Review = "Even better on rewatch."
	updatedEntry, err := watchEntries.Upsert(ctx, entry)
	if err != nil {
		t.Fatalf("update watch entry: %v", err)
	}
	if updatedEntry.ID != entry.ID || updatedEntry.Rating == nil || *updatedEntry.Rating != 9 {
		t.Fatalf("unexpected updated watch entry: %+v", updatedEntry)
	}

	if err := movies.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete movie: %v", err)
	}

	_, err = movies.GetByID(ctx, created.ID)
	if !errors.Is(err, apperrors.ErrMovieNotFound) {
		t.Fatalf("expected ErrMovieNotFound, got %v", err)
	}

	_, err = watchEntries.GetByMovieID(ctx, created.ID)
	if !errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		t.Fatalf("expected cascaded watch entry delete, got %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.OpenAndMigrate(
		"file:movietracker_repository_test?mode=memory&cache=shared&_pragma=foreign_keys(1)",
		database.ClientMigrations,
		"migrations/client",
	)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	return db
}
