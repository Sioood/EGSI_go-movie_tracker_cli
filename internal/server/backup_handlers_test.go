package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/movietracker/movie-tracker/internal/config"
)

func TestBackupExportDefaultsWhenEmpty(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "backup-empty")

	rr := authGet(t, router, "/api/v1/backup/config", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("export config: want 200, got %d: %s", rr.Code, rr.Body)
	}
	var cfg config.Config
	if err := json.NewDecoder(rr.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Theme != "midnight" {
		t.Fatalf("expected default theme, got %s", cfg.Theme)
	}

	rr = authGet(t, router, "/api/v1/backup/state", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("export state: want 200, got %d", rr.Code)
	}
}

func TestBackupRoundTripSnapshot(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "backup-roundtrip")

	payload := map[string]any{
		"config": map[string]any{
			"theme":        "solar",
			"server_url":   "http://prod.example.com",
			"offline_mode": false,
		},
		"state": map[string]any{
			"last_route": "movie_list",
			"filter":     "watched",
			"sort":       "rating",
		},
	}
	rr := authPut(t, router, "/api/v1/backup", token, payload)
	if rr.Code != http.StatusOK {
		t.Fatalf("import snapshot: want 200, got %d: %s", rr.Code, rr.Body)
	}

	rr = authGet(t, router, "/api/v1/backup", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("export snapshot: want 200, got %d: %s", rr.Code, rr.Body)
	}

	var exported struct {
		Config config.Config `json:"config"`
		State  config.State  `json:"state"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&exported); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if exported.Config.Theme != "solar" || exported.Config.ServerURL != "http://prod.example.com" {
		t.Fatalf("unexpected config: %+v", exported.Config)
	}
	if exported.State.LastRoute != "movie_list" || exported.State.Filter != "watched" {
		t.Fatalf("unexpected state: %+v", exported.State)
	}
}

func TestBackupCrossUserIsolation(t *testing.T) {
	router := newFullRouter(t)
	tokenA := registerAndLogin(t, router, "backup-a")
	tokenB := registerAndLogin(t, router, "backup-b")

	rr := authPut(t, router, "/api/v1/backup/config", tokenA, map[string]any{
		"theme":        "forest",
		"server_url":   "http://a.example.com",
		"offline_mode": true,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("import config A: %d", rr.Code)
	}

	rr = authGet(t, router, "/api/v1/backup/config", tokenB)
	if rr.Code != http.StatusOK {
		t.Fatalf("export config B: %d", rr.Code)
	}
	var cfgB config.Config
	if err := json.NewDecoder(rr.Body).Decode(&cfgB); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfgB.Theme == "forest" && cfgB.ServerURL == "http://a.example.com" {
		t.Fatal("user B should not see user A backup")
	}
}
