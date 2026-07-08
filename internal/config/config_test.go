package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/movietracker/movie-tracker/internal/config"
)

func TestDefaultConfigWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := config.LoadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	defaults := config.DefaultConfig()
	if cfg.Theme != defaults.Theme || cfg.ServerURL != defaults.ServerURL || cfg.OfflineMode != defaults.OfflineMode {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected no config file created on load")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	cfg := config.Config{
		Theme:       "solar",
		ServerURL:   "http://127.0.0.1:9090",
		OfflineMode: false,
	}
	if err := config.SaveConfigToDir(dir, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := config.LoadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded != cfg {
		t.Fatalf("round-trip mismatch: got %+v want %+v", loaded, cfg)
	}

	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("want 0600, got %o", info.Mode().Perm())
	}
}

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()

	state := config.State{
		LastRoute:  "movie_list",
		Filter:     "watched",
		Sort:       "rating",
		LastSyncAt: "2026-07-08T12:00:00Z",
	}
	if err := config.SaveStateToDir(dir, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	loaded, err := config.LoadStateFromDir(dir)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if loaded != state {
		t.Fatalf("round-trip mismatch: got %+v want %+v", loaded, state)
	}
}

func TestSessionRoundTripAndClear(t *testing.T) {
	dir := t.TempDir()

	sess := config.Session{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ServerUserID: "user-1",
		Email:        "alice@example.com",
	}
	if err := config.SaveSessionToDir(dir, sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := config.LoadSessionFromDir(dir)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded != sess {
		t.Fatalf("round-trip mismatch: got %+v want %+v", loaded, sess)
	}

	if err := config.ClearSessionInDir(dir); err != nil {
		t.Fatalf("clear session: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "session.json")); !os.IsNotExist(err) {
		t.Fatal("expected session file removed")
	}
}

func TestSaveEmptySessionRemovesFile(t *testing.T) {
	dir := t.TempDir()

	if err := config.SaveSessionToDir(dir, config.Session{
		AccessToken:  "a",
		RefreshToken: "r",
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := config.SaveSessionToDir(dir, config.Session{}); err != nil {
		t.Fatalf("clear via empty save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "session.json")); !os.IsNotExist(err) {
		t.Fatal("expected session file removed on empty save")
	}
}

func TestLegacyYAMLMigration(t *testing.T) {
	home := t.TempDir()
	legacyDir := filepath.Join(home, ".movietracker")
	if err := os.MkdirAll(legacyDir, 0o700); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	legacyConfig := "theme: forest\nserver_url: http://legacy:9000\noffline_mode: false\n"
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	dir, err := config.Dir()
	if err != nil {
		t.Fatalf("dir: %v", err)
	}

	cfg, err := config.LoadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
	if cfg.Theme != "forest" || cfg.ServerURL != "http://legacy:9000" || cfg.OfflineMode {
		t.Fatalf("unexpected migrated config: %+v", cfg)
	}
	if _, err := os.Stat(legacyDir + ".migrated"); err != nil {
		t.Fatalf("expected archived legacy dir: %v", err)
	}
}
