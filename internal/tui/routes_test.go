package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

type fakeSyncRunner struct {
	runResult     SyncResult
	runErr        error
	pendingCount  int
	conflictCount int
	runCalls      int
}

func (f *fakeSyncRunner) Run(ctx context.Context) (SyncResult, error) {
	f.runCalls++
	if f.runErr != nil {
		return SyncResult{}, f.runErr
	}
	return f.runResult, nil
}

func (f *fakeSyncRunner) PendingCount(ctx context.Context) (int, error) {
	return f.pendingCount, nil
}

func (f *fakeSyncRunner) ConflictCount(ctx context.Context) (int, error) {
	return f.conflictCount, nil
}

func (f *fakeSyncRunner) ListConflicts(ctx context.Context) ([]domain.SyncConflict, error) {
	return nil, nil
}

func (f *fakeSyncRunner) ResolveConflict(ctx context.Context, id, choice string) error {
	return nil
}

func (f *fakeSyncRunner) GetDeviceName(ctx context.Context, deviceID string) (string, error) {
	return deviceID, nil
}

func TestStartSyncUpdatesStatus(t *testing.T) {
	runner := &fakeSyncRunner{
		runResult: SyncResult{PendingCount: 0, ConflictCount: 0},
	}
	model := New(Options{
		MovieService: newFakeMovieService(),
		SyncRunner:   runner,
		State: AppState{
			Config:  Config{Theme: "midnight", OfflineMode: false},
			Session: SessionState{Authenticated: true, AccessToken: "token"},
		},
	})

	updated, cmd := model.startSync()
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updated)
	}
	if !m.syncSyncing || m.syncStatus != SyncStatusSyncing {
		t.Fatalf("expected syncing state, got status=%s syncing=%v", m.syncStatus, m.syncSyncing)
	}
	if cmd == nil {
		t.Fatal("expected sync command")
	}

	msg := cmd()
	result, ok := msg.(syncResultMsg)
	if !ok {
		t.Fatalf("expected syncResultMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected sync error: %v", result.err)
	}

	final, _ := m.handleSyncResult(result)
	fm := final.(Model)
	if fm.syncStatus != SyncStatusSynced || fm.syncSyncing {
		t.Fatalf("expected synced idle state, got status=%s syncing=%v", fm.syncStatus, fm.syncSyncing)
	}
	if runner.runCalls != 1 {
		t.Fatalf("expected one sync run, got %d", runner.runCalls)
	}
}

func TestStartSyncSkippedWhenOffline(t *testing.T) {
	runner := &fakeSyncRunner{}
	model := New(Options{
		MovieService: newFakeMovieService(),
		SyncRunner:   runner,
		State: AppState{
			Config:  Config{OfflineMode: true},
			Session: SessionState{Authenticated: true, AccessToken: "token"},
		},
	})

	updated, cmd := model.startSync()
	m := updated.(Model)
	if m.syncSyncing || cmd != nil {
		t.Fatal("expected sync to be skipped in offline mode")
	}
	if runner.runCalls != 0 {
		t.Fatalf("expected no sync run, got %d", runner.runCalls)
	}
}

func TestSyncResultErrorSetsPendingCount(t *testing.T) {
	runner := &fakeSyncRunner{
		runErr:       apperrors.ErrNetwork,
		pendingCount: 3,
	}
	model := New(Options{
		MovieService: newFakeMovieService(),
		SyncRunner:   runner,
		State: AppState{
			Config:  Config{Theme: "midnight", OfflineMode: false},
			Session: SessionState{Authenticated: true, AccessToken: "token"},
		},
	})
	model.syncSyncing = true

	final, cmd := model.handleSyncResult(syncResultMsg{err: apperrors.ErrNetwork})
	fm := final.(Model)
	if fm.syncStatus != SyncStatusError || fm.pendingCount != 3 {
		t.Fatalf("expected error with pending=3, got status=%s pending=%d", fm.syncStatus, fm.pendingCount)
	}
	if cmd == nil {
		t.Fatal("expected follow-up tick command")
	}
}

func TestCtrlCShutdownCancelsContext(t *testing.T) {
	model := testNewModel(newFakeMovieService())
	if model.ctx == nil || model.cancel == nil {
		t.Fatal("expected root context on model")
	}

	updated, cmd := model.handleKey(keyMsg("ctrl+c"))
	m := updated.(Model)
	m.shutdown()
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.Quit, got %T", cmd())
	}

	select {
	case <-m.ctx.Done():
	default:
		t.Fatal("expected context cancelled after shutdown")
	}
}

func TestMovieDeleteFromList(t *testing.T) {
	store := newFakeMovieService()
	store.movies = []domain.Movie{{ID: "movie-1", UserID: "local-user", Title: "Heat", Year: 1995}}
	model := testNewModel(store)
	model.goTo(RouteMovieList)
	model.refreshMovies()

	model = press(t, model, "d")
	if len(store.movies) != 0 {
		t.Fatalf("expected movie deleted, got %d movies", len(store.movies))
	}
	if model.messageKind != messages.KindSuccess {
		t.Fatalf("expected success message, got %s", model.message)
	}
}

func TestLoginCmdUsesAuthClient(t *testing.T) {
	auth := &fakeAuthClient{
		pair: service.TokenPair{AccessToken: "access", RefreshToken: "refresh"},
		user: UserInfo{ID: "user-1", Email: "alice@example.com"},
	}
	model := testNewModelWithAuth(newFakeMovieService(), auth, nil)

	msg := model.loginCmd("alice@example.com", "secret123")()
	result, ok := msg.(authResultMsg)
	if !ok || result.err != nil {
		t.Fatalf("expected login success, got %+v ok=%v", result, ok)
	}
	if result.session.Email != "alice@example.com" {
		t.Fatalf("unexpected session: %+v", result.session)
	}
}
