package client_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
)

func newSyncTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, t.Name())

	dsn := "file:" + name + "?mode=memory&cache=shared&_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ServerMigrations, "migrations/server")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	secret := []byte("phase-9-sync-client-test-secret")
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, secret)
	movieRepo := repository.NewMovieRepository(db)
	watchRepo := repository.NewWatchEntryRepository(db)
	movieSvc := service.NewMovieService(movieRepo, watchRepo)
	statsRepo := repository.NewStatsRepository(db)
	statsSvc := service.NewStatsService(statsRepo)
	router := server.NewRouter(server.Services{Auth: authSvc, Movies: movieSvc, Stats: statsSvc}, secret)
	return httptest.NewServer(router)
}

func TestSyncClientRoundtrip(t *testing.T) {
	srv := newSyncTestServer(t)
	defer srv.Close()

	auth := client.NewAuthClient(srv.URL)
	syncClient := client.NewSyncClient(srv.URL)
	ctx := context.Background()

	pair, err := auth.Register(ctx, "sync@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	now := time.Now().UTC()
	importResult, err := syncClient.Import(ctx, pair.AccessToken, client.SyncPayload{
		Movies: []domain.Movie{{
			ID:        "movie-sync-1",
			UserID:    "ignored",
			Title:     "Synced Film",
			Year:      2024,
			UpdatedAt: now,
			CreatedAt: now,
		}},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if importResult.SyncedMovies != 1 {
		t.Fatalf("expected 1 synced movie, got %d", importResult.SyncedMovies)
	}

	exported, err := syncClient.Export(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(exported.Movies) != 1 || exported.Movies[0].Title != "Synced Film" {
		t.Fatalf("unexpected export: %+v", exported.Movies)
	}

	importResult, err = syncClient.Import(ctx, pair.AccessToken, client.SyncPayload{
		DeletedMovieIDs: []string{"movie-sync-1"},
	})
	if err != nil {
		t.Fatalf("import delete: %v", err)
	}
	if importResult.DeletedMovies != 1 {
		t.Fatalf("expected 1 deleted movie, got %d", importResult.DeletedMovies)
	}
}
