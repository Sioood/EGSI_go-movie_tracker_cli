package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/movietracker/movie-tracker/internal/config"
)

func TestDefaultConfigWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

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

	info, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("want 0600, got %o", info.Mode().Perm())
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
	if _, err := os.Stat(filepath.Join(dir, "session.yaml")); !os.IsNotExist(err) {
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
	if _, err := os.Stat(filepath.Join(dir, "session.yaml")); !os.IsNotExist(err) {
		t.Fatal("expected session file removed on empty save")
	}
}
