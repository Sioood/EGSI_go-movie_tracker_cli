package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/service"
)

func openSyncUnitDB(t *testing.T) (*sql.DB, *repository.SyncRepository, *service.MovieService) {
	t.Helper()

	db, err := database.OpenAndMigrate(
		"file:sync_unit_"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)",
		database.ClientMigrations,
		"migrations/client",
	)
	if err != nil {
		t.Fatalf("open sync unit db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	movieRepo := repository.NewMovieRepository(db)
	watchRepo := repository.NewWatchEntryRepository(db)
	syncRepo := repository.NewSyncRepository(db, movieRepo, watchRepo)
	return db, syncRepo, service.NewMovieService(movieRepo, watchRepo)
}

func TestApplyRemoteMovieRecordsConflict(t *testing.T) {
	_, syncRepo, movies := openSyncUnitDB(t)
	svc := &Service{
		movies:   movies,
		syncRepo: syncRepo,
	}

	ctx := context.Background()
	lastSync := time.Now().UTC().Add(-3 * time.Hour)
	meta := repository.SyncMetadata{LastSyncAt: &lastSync}
	localUpdated := time.Now().UTC().Add(-2 * time.Hour)
	remoteUpdated := time.Now().UTC().Add(-1 * time.Hour)

	local := domain.Movie{
		ID:              "movie-conflict-1",
		UserID:          LocalUserID,
		Title:           "Inception",
		Year:            2010,
		UpdatedAt:       localUpdated,
		UpdatedByDevice: "device-a",
	}
	if _, _, err := movies.SyncUpsertMovie(ctx, LocalUserID, local); err != nil {
		t.Fatalf("seed local movie: %v", err)
	}

	remote := domain.Movie{
		ID:              "movie-conflict-1",
		UserID:          LocalUserID,
		Title:           "Inception Remastered",
		Year:            2010,
		UpdatedAt:       remoteUpdated,
		UpdatedByDevice: "device-b",
	}

	applied, err := svc.applyRemoteMovie(ctx, meta, remote)
	if err != nil {
		t.Fatalf("apply remote movie: %v", err)
	}
	if applied {
		t.Fatal("expected conflict instead of apply")
	}

	count, err := syncRepo.ConflictCount(ctx)
	if err != nil {
		t.Fatalf("conflict count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 conflict, got %d", count)
	}
}

func TestApplyRemoteMovieAppliesWhenNoConflict(t *testing.T) {
	_, syncRepo, movies := openSyncUnitDB(t)
	svc := &Service{
		movies:   movies,
		syncRepo: syncRepo,
	}

	ctx := context.Background()
	meta := repository.SyncMetadata{}
	remoteUpdated := time.Now().UTC()

	remote := domain.Movie{
		ID:              "movie-new-1",
		UserID:          LocalUserID,
		Title:           "Heat",
		Year:            1995,
		UpdatedAt:       remoteUpdated,
		UpdatedByDevice: "device-b",
	}

	applied, err := svc.applyRemoteMovie(ctx, meta, remote)
	if err != nil {
		t.Fatalf("apply remote movie: %v", err)
	}
	if !applied {
		t.Fatal("expected remote movie to be applied")
	}

	got, err := movies.GetMovie(ctx, "movie-new-1")
	if err != nil || got.Title != "Heat" {
		t.Fatalf("expected imported movie, got %+v err=%v", got, err)
	}
}

func TestApplyRemoteWatchEntryRecordsConflict(t *testing.T) {
	db, syncRepo, movies := openSyncUnitDB(t)
	watchRepo := repository.NewWatchEntryRepository(db)
	svc := &Service{
		movies:   movies,
		syncRepo: syncRepo,
	}

	ctx := context.Background()
	lastSync := time.Now().UTC().Add(-3 * time.Hour)
	meta := repository.SyncMetadata{LastSyncAt: &lastSync}
	localUpdated := time.Now().UTC().Add(-2 * time.Hour)
	remoteUpdated := time.Now().UTC().Add(-1 * time.Hour)

	movie, err := movies.CreateMovie(ctx, domain.Movie{UserID: LocalUserID, Title: "Arrival", Year: 2016})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	localRating := 8.0
	remoteRating := 9.0
	local := domain.WatchEntry{
		MovieID:         movie.ID,
		Watched:         true,
		Rating:          &localRating,
		RatingScale:     10,
		UpdatedAt:       localUpdated,
		UpdatedByDevice: "device-a",
	}
	if _, _, err := watchRepo.SyncUpsert(ctx, local); err != nil {
		t.Fatalf("seed local entry: %v", err)
	}

	remote := domain.WatchEntry{
		MovieID:         movie.ID,
		Watched:         true,
		Rating:          &remoteRating,
		RatingScale:     10,
		UpdatedAt:       remoteUpdated,
		UpdatedByDevice: "device-b",
	}

	applied, err := svc.applyRemoteWatchEntry(ctx, meta, remote)
	if err != nil {
		t.Fatalf("apply remote watch entry: %v", err)
	}
	if applied {
		t.Fatal("expected conflict instead of apply")
	}

	count, err := syncRepo.ConflictCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 conflict, got %d err=%v", count, err)
	}
}

func TestShouldRecordConflictWhenPending(t *testing.T) {
	_, syncRepo, _ := openSyncUnitDB(t)
	ctx := context.Background()
	lastSync := time.Now().UTC().Add(-1 * time.Hour)
	meta := repository.SyncMetadata{LastSyncAt: &lastSync}

	if err := syncRepo.MarkPending(ctx, repository.PendingEntityMovie, "movie-pending", repository.PendingOpUpsert); err != nil {
		t.Fatalf("mark pending: %v", err)
	}

	if !shouldRecordConflict(ctx, syncRepo, meta, domain.SyncEntityMovie, "movie-pending", time.Now().UTC(), time.Now().UTC()) {
		t.Fatal("expected conflict recording when entity is pending")
	}
}

func TestShouldRecordConflictSkipsBeforeLastSync(t *testing.T) {
	_, syncRepo, _ := openSyncUnitDB(t)
	ctx := context.Background()
	lastSync := time.Now().UTC()
	meta := repository.SyncMetadata{LastSyncAt: &lastSync}
	before := lastSync.Add(-time.Hour)

	if shouldRecordConflict(ctx, syncRepo, meta, domain.SyncEntityMovie, "movie-old", before, before) {
		t.Fatal("expected no conflict when both sides updated before last sync")
	}
}
