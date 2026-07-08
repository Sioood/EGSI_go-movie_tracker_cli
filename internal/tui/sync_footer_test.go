package tui

import (
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

func TestSyncFooterOffline(t *testing.T) {
	m := testNewModel(newFakeMovieService())
	m.state.Config.OfflineMode = true

	line := m.syncFooterLine()
	if line != messages.UI.SyncOffline {
		t.Fatalf("want offline message, got %q", line)
	}
}

func TestSyncFooterSyncing(t *testing.T) {
	m := testNewModel(newFakeMovieService())
	m.state.Config.OfflineMode = false
	m.syncStatus = SyncStatusSyncing

	line := m.syncFooterLine()
	if line != messages.UI.SyncSyncing {
		t.Fatalf("want syncing message, got %q", line)
	}
}

func TestSyncFooterPending(t *testing.T) {
	m := testNewModel(newFakeMovieService())
	m.state.Config.OfflineMode = false
	m.pendingCount = 3
	m.syncStatus = SyncStatusPending

	line := m.syncFooterLine()
	if !strings.Contains(line, "3") {
		t.Fatalf("want pending count in footer, got %q", line)
	}
}

func TestSyncFooterError(t *testing.T) {
	m := testNewModel(newFakeMovieService())
	m.state.Config.OfflineMode = false
	m.syncStatus = SyncStatusError
	m.syncError = "network down"

	line := m.syncFooterLine()
	if !strings.Contains(line, messages.UI.SyncError) {
		t.Fatalf("want error prefix, got %q", line)
	}
	if !strings.Contains(line, "network down") {
		t.Fatalf("want error detail, got %q", line)
	}
}

func TestLogoutResetsSyncState(t *testing.T) {
	store := newFakeMovieService()
	m := testNewModel(store)
	m.goTo(RouteSettings)
	m.state.Session = SessionState{Authenticated: true, Email: "a@example.com", RefreshToken: "rt"}
	m.syncStatus = SyncStatusError
	m.syncError = "sync failed"
	m.pendingCount = 5

	updated, _ := m.updateSettings(keyMsg("d"))
	m = updated.(Model)

	if m.state.Session.Authenticated {
		t.Fatal("expected session cleared after logout")
	}
	if m.syncStatus != SyncStatusIdle {
		t.Fatalf("want idle sync status, got %q", m.syncStatus)
	}
	if m.syncError != "" {
		t.Fatalf("want empty sync error, got %q", m.syncError)
	}
	if m.pendingCount != 0 {
		t.Fatalf("want pending count 0, got %d", m.pendingCount)
	}
}
