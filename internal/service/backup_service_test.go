package service

import (
	"context"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
)

func openBackupTestDB(t *testing.T) (*repository.BackupRepository, string) {
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
		t.Fatalf("open backup test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	userID := "backup-user-" + name
	users := repository.NewUserRepository(db)
	if _, err := users.Create(context.Background(), domain.User{
		ID:           userID,
		Email:        userID + "@example.com",
		PasswordHash: "hash",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return repository.NewBackupRepository(db), userID
}

func TestBackupServiceRoundTripSnapshot(t *testing.T) {
	repo, userID := openBackupTestDB(t)
	svc := NewBackupService(repo)
	ctx := context.Background()

	snapshot := BackupSnapshot{
		Config: config.Config{Theme: "solar", ServerURL: "http://example.com", OfflineMode: false},
		State:  config.State{LastRoute: "movie_list", Filter: "watched", Sort: "rating"},
	}
	if err := svc.ImportSnapshot(ctx, userID, snapshot); err != nil {
		t.Fatalf("import snapshot: %v", err)
	}

	exported, err := svc.ExportSnapshot(ctx, userID)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	if exported.Config.Theme != "solar" || exported.Config.ServerURL != "http://example.com" {
		t.Fatalf("unexpected config: %+v", exported.Config)
	}
	if exported.State.LastRoute != "movie_list" || exported.State.Filter != "watched" {
		t.Fatalf("unexpected state: %+v", exported.State)
	}
}

func TestBackupServiceRejectsCorruptConfigJSON(t *testing.T) {
	repo, userID := openBackupTestDB(t)
	ctx := context.Background()

	if err := repo.UpsertConfig(ctx, userID, `{not-json`); err != nil {
		t.Fatalf("seed corrupt config: %v", err)
	}

	svc := NewBackupService(repo)
	_, err := svc.ExportConfig(ctx, userID)
	if err == nil {
		t.Fatal("expected error for corrupt config JSON")
	}
	if !strings.Contains(err.Error(), "parse backup config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupServiceRejectsCorruptStateJSON(t *testing.T) {
	repo, userID := openBackupTestDB(t)
	ctx := context.Background()

	if err := repo.UpsertState(ctx, userID, `{broken`); err != nil {
		t.Fatalf("seed corrupt state: %v", err)
	}

	svc := NewBackupService(repo)
	_, err := svc.ExportState(ctx, userID)
	if err == nil {
		t.Fatal("expected error for corrupt state JSON")
	}
	if !strings.Contains(err.Error(), "parse backup state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupServiceExportDefaultsWhenEmpty(t *testing.T) {
	repo, userID := openBackupTestDB(t)
	svc := NewBackupService(repo)
	ctx := context.Background()

	cfg, err := svc.ExportConfig(ctx, userID)
	if err != nil {
		t.Fatalf("export config: %v", err)
	}
	if cfg.Theme != "midnight" {
		t.Fatalf("expected default theme, got %s", cfg.Theme)
	}

	state, err := svc.ExportState(ctx, userID)
	if err != nil {
		t.Fatalf("export state: %v", err)
	}
	if state.LastRoute != "splash" || state.Filter != "all" {
		t.Fatalf("expected default state, got %+v", state)
	}
}

func TestBackupServiceStripsTMDBKeyOnImport(t *testing.T) {
	repo, userID := openBackupTestDB(t)
	svc := NewBackupService(repo)
	ctx := context.Background()

	cfg := config.Config{Theme: "forest", TMDBAPIKey: "super-secret"}
	if err := svc.ImportConfig(ctx, userID, cfg); err != nil {
		t.Fatalf("import config: %v", err)
	}

	exported, err := svc.ExportConfig(ctx, userID)
	if err != nil {
		t.Fatalf("export config: %v", err)
	}
	if exported.TMDBAPIKey != "" {
		t.Fatalf("expected TMDB key stripped, got %q", exported.TMDBAPIKey)
	}
}
