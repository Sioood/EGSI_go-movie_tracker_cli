package sync_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
	appsync "github.com/movietracker/movie-tracker/internal/sync"
)

func TestServiceRunMigratesAndSyncs(t *testing.T) {
	router := newSyncTestRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	clientDB := openClientTestDB(t)
	defer clientDB.Close()

	movieRepo := repository.NewMovieRepository(clientDB)
	watchRepo := repository.NewWatchEntryRepository(clientDB)
	syncRepo := repository.NewSyncRepository(clientDB, movieRepo, watchRepo)
	movieService := service.NewMovieService(movieRepo, watchRepo)

	ctx := context.Background()
	created, err := movieService.CreateMovie(ctx, domain.Movie{
		UserID: appsync.LocalUserID,
		Title:  "Offline Film",
		Year:   2021,
	})
	if err != nil {
		t.Fatalf("create local movie: %v", err)
	}
	if err := syncRepo.MarkPending(ctx, repository.PendingEntityMovie, created.ID, repository.PendingOpUpsert); err != nil {
		t.Fatalf("mark pending: %v", err)
	}

	authClient := client.NewAuthClient(srv.URL)
	pair, err := authClient.Register(ctx, "hybrid@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	info, err := authClient.Me(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("me: %v", err)
	}

	session := appsync.SessionAccess{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ServerUserID: info.ID,
	}
	syncService := appsync.NewService(
		movieService,
		syncRepo,
		client.NewSyncClient(srv.URL),
		&tokenRefresher{auth: authClient},
		func() appsync.SessionAccess { return session },
		func() string { return "test-device" },
		func() bool { return true },
		nil,
	)

	result, err := syncService.Run(ctx)
	if err != nil {
		t.Fatalf("sync run: %v", err)
	}
	if result.PushedMovies == 0 {
		t.Fatalf("expected pushed movies, got %+v", result)
	}

	meta, err := syncRepo.GetMetadata(ctx)
	if err != nil || !meta.UserIDMigrated {
		t.Fatalf("expected migrated metadata, got %+v err=%v", meta, err)
	}

	userID, err := syncService.UserID(ctx)
	if err != nil || userID != info.ID {
		t.Fatalf("expected server user id %s, got %s err=%v", info.ID, userID, err)
	}
}

type tokenRefresher struct {
	auth *client.AuthClient
}

func (t *tokenRefresher) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	pair, err := t.auth.Refresh(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}
	return pair.AccessToken, pair.RefreshToken, nil
}

func newSyncTestRouter(t *testing.T) http.Handler {
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
		t.Fatalf("open server db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	secret := []byte("phase-9-sync-test-secret")
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, secret)
	movieRepo := repository.NewMovieRepository(db)
	watchRepo := repository.NewWatchEntryRepository(db)
	movieSvc := service.NewMovieService(movieRepo, watchRepo)
	return server.NewRouter(server.Services{Auth: authSvc, Movies: movieSvc}, secret)
}

func openClientTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.OpenAndMigrate(
		"file:phase9_client_sync?mode=memory&cache=shared&_pragma=foreign_keys(1)",
		database.ClientMigrations,
		"migrations/client",
	)
	if err != nil {
		t.Fatalf("open client db: %v", err)
	}
	return db
}
