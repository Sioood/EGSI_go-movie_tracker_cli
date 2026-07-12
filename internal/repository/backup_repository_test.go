package repository

import (
	"context"
	"testing"
)

func TestBackupRepositoryRoundTrip(t *testing.T) {
	db := openServerTestDB(t)
	seedServerUser(t, db, "user-1", "user1@example.com")
	repo := NewBackupRepository(db)
	ctx := context.Background()
	userID := "user-1"

	empty, err := repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get empty backup: %v", err)
	}
	if empty.Config != "{}" || empty.State != "{}" {
		t.Fatalf("expected empty defaults, got %+v", empty)
	}

	configJSON := `{"theme":"solar","server_url":"http://example.com","offline_mode":false}`
	if err := repo.UpsertConfig(ctx, userID, configJSON); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	stateJSON := `{"last_route":"movie_list","filter":"watched"}`
	if err := repo.UpsertState(ctx, userID, stateJSON); err != nil {
		t.Fatalf("upsert state: %v", err)
	}

	got, err := repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get backup: %v", err)
	}
	if got.Config != configJSON {
		t.Fatalf("config mismatch: %s", got.Config)
	}
	if got.State != stateJSON {
		t.Fatalf("state mismatch: %s", got.State)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at to be set")
	}
}

func TestBackupRepositoryUpsertFull(t *testing.T) {
	db := openServerTestDB(t)
	seedServerUser(t, db, "user-full", "full@example.com")
	repo := NewBackupRepository(db)
	ctx := context.Background()
	userID := "user-full"

	configJSON := `{"theme":"forest"}`
	stateJSON := `{"sort":"rating"}`
	if err := repo.UpsertFull(ctx, userID, configJSON, stateJSON); err != nil {
		t.Fatalf("upsert full: %v", err)
	}

	got, err := repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Config != configJSON || got.State != stateJSON {
		t.Fatalf("unexpected backup: %+v", got)
	}

	updatedConfig := `{"theme":"midnight"}`
	if err := repo.UpsertFull(ctx, userID, updatedConfig, stateJSON); err != nil {
		t.Fatalf("upsert full update: %v", err)
	}
	got, err = repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Config != updatedConfig {
		t.Fatalf("expected updated config, got %s", got.Config)
	}
}

func TestBackupRepositoryUpsertFullRequiresUserID(t *testing.T) {
	db := openServerTestDB(t)
	repo := NewBackupRepository(db)
	err := repo.UpsertFull(context.Background(), "", "{}", "{}")
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
}
