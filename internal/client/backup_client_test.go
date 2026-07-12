package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
)

func newFullTestServer(t *testing.T) *httptest.Server {
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

	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)
	watchRepo := repository.NewWatchEntryRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	backupRepo := repository.NewBackupRepository(db)

	router := server.NewRouter(server.Services{
		Auth:    service.NewAuthService(userRepo, testSecret),
		Movies:  service.NewMovieService(movieRepo, watchRepo),
		Stats:   service.NewStatsService(statsRepo),
		Backups: service.NewBackupService(backupRepo),
	}, testSecret)
	return httptest.NewServer(router)
}

func registerBackupClientToken(t *testing.T, srvURL string) string {
	t.Helper()
	authClient := client.NewAuthClient(srvURL)
	pair, err := authClient.Register(context.Background(), "backup-client@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return pair.AccessToken
}

func TestBackupClientRoundTripSnapshot(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	token := registerBackupClientToken(t, srv.URL)
	c := client.NewBackupClient(srv.URL)
	ctx := context.Background()

	snapshot := service.BackupSnapshot{
		Config: config.Config{Theme: "solar", ServerURL: "http://prod.example.com"},
		State:  config.State{LastRoute: "settings", Filter: "watched"},
	}
	if err := c.ImportSnapshot(ctx, token, snapshot); err != nil {
		t.Fatalf("import snapshot: %v", err)
	}

	exported, err := c.ExportSnapshot(ctx, token)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	if exported.Config.Theme != "solar" || exported.State.LastRoute != "settings" {
		t.Fatalf("unexpected snapshot: config=%+v state=%+v", exported.Config, exported.State)
	}
}

func TestBackupClientUnauthorized(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	c := client.NewBackupClient(srv.URL)
	_, err := c.ExportSnapshot(context.Background(), "invalid-token")
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestBackupClientExportConfigAndState(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	token := registerBackupClientToken(t, srv.URL)
	c := client.NewBackupClient(srv.URL)
	ctx := context.Background()

	cfg := config.Config{Theme: "forest", ServerURL: "http://cli.example.com"}
	if err := c.ImportConfig(ctx, token, cfg); err != nil {
		t.Fatalf("import config: %v", err)
	}
	exportedCfg, err := c.ExportConfig(ctx, token)
	if err != nil {
		t.Fatalf("export config: %v", err)
	}
	if exportedCfg.Theme != "forest" {
		t.Fatalf("unexpected config theme: %s", exportedCfg.Theme)
	}

	state := config.State{Sort: "rating"}
	if err := c.ImportState(ctx, token, state); err != nil {
		t.Fatalf("import state: %v", err)
	}
	exportedState, err := c.ExportState(ctx, token)
	if err != nil {
		t.Fatalf("export state: %v", err)
	}
	if exportedState.Sort != "rating" {
		t.Fatalf("unexpected state sort: %s", exportedState.Sort)
	}
}

func TestBackupClientSetBaseURL(t *testing.T) {
	c := client.NewBackupClient("http://localhost:8080/")
	c.SetBaseURL("http://example.com")
	if c.BaseURL != "http://example.com" {
		t.Fatalf("expected normalized base URL, got %q", c.BaseURL)
	}
}

func TestBackupClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
	}))
	defer srv.Close()

	c := client.NewBackupClient(srv.URL)
	err := c.ImportConfig(context.Background(), "token", config.Config{Theme: "midnight"})
	if err == nil {
		t.Fatal("expected API error")
	}
}
