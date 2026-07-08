package repository

import (
	"context"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestSyncRepositoryMetadataAndPending(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	entries := NewWatchEntryRepository(db)
	syncRepo := NewSyncRepository(db, movies, entries)

	meta, err := syncRepo.GetMetadata(ctx)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if meta.UserIDMigrated {
		t.Fatal("expected fresh metadata not migrated")
	}

	if err := syncRepo.MarkPending(ctx, PendingEntityMovie, "movie-1", PendingOpUpsert); err != nil {
		t.Fatalf("mark pending: %v", err)
	}
	count, err := syncRepo.PendingCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("pending count: %d err=%v", count, err)
	}

	now := time.Now().UTC()
	meta.LastSyncAt = &now
	meta.UserIDMigrated = true
	if err := syncRepo.UpdateMetadata(ctx, meta); err != nil {
		t.Fatalf("update metadata: %v", err)
	}

	loaded, err := syncRepo.GetMetadata(ctx)
	if err != nil || !loaded.UserIDMigrated || loaded.LastSyncAt == nil {
		t.Fatalf("unexpected metadata: %+v err=%v", loaded, err)
	}

	if err := syncRepo.ClearPending(ctx, PendingEntityMovie, "movie-1"); err != nil {
		t.Fatalf("clear pending: %v", err)
	}
	count, err = syncRepo.PendingCount(ctx)
	if err != nil || count != 0 {
		t.Fatalf("expected zero pending, got %d", count)
	}
}

func TestMovieSyncUpsertLWW(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)

	base := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	movie := domain.Movie{
		UserID:    "user-1",
		Title:     "Original",
		Year:      2020,
		CreatedAt: base,
		UpdatedAt: base,
	}
	saved, _, err := movies.SyncUpsert(ctx, movie)
	if err != nil {
		t.Fatalf("seed sync upsert: %v", err)
	}

	stale := saved
	stale.Title = "Stale"
	stale.UpdatedAt = base.Add(-time.Hour)
	if _, applied, err := movies.SyncUpsert(ctx, stale); err != nil || applied {
		t.Fatalf("stale sync should be skipped, applied=%t err=%v", applied, err)
	}

	got, err := movies.GetByID(ctx, saved.ID)
	if err != nil || got.Title != "Original" {
		t.Fatalf("unexpected title after stale sync: %+v err=%v", got, err)
	}

	fresh := got
	fresh.Title = "Fresh"
	fresh.UpdatedAt = base.Add(time.Hour)
	if _, applied, err := movies.SyncUpsert(ctx, fresh); err != nil || !applied {
		t.Fatalf("fresh sync should apply, applied=%t err=%v", applied, err)
	}

	got, err = movies.GetByID(ctx, saved.ID)
	if err != nil || got.Title != "Fresh" {
		t.Fatalf("expected fresh title, got %+v", got)
	}
}

func TestSyncRepositoryMigrateUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	syncRepo := NewSyncRepository(db, movies, NewWatchEntryRepository(db))

	if _, err := movies.Create(ctx, domain.Movie{UserID: "local-user", Title: "Local", Year: 2020}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := syncRepo.MigrateUserID(ctx, "local-user", "server-user"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	list, err := movies.ListByUser(ctx, "server-user")
	if err != nil || len(list) != 1 {
		t.Fatalf("expected migrated movie, got %+v err=%v", list, err)
	}
}
