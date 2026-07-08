package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

const syncInterval = 30 * time.Second

// SyncStatus describes the footer sync indicator state.
type SyncStatus string

const (
	SyncStatusIdle    SyncStatus = "idle"
	SyncStatusSyncing SyncStatus = "syncing"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusPending SyncStatus = "pending"
	SyncStatusError   SyncStatus = "error"
)

// SyncResult is returned by a completed sync run.
type SyncResult struct {
	PendingCount int
}

// SyncRunner performs background synchronization.
type SyncRunner interface {
	Run(ctx context.Context) (SyncResult, error)
	PendingCount(ctx context.Context) (int, error)
}

// SyncRequestMsg asks the model to start a sync when online.
type SyncRequestMsg struct{}

type syncTickMsg struct{}
type syncResultMsg struct {
	result SyncResult
	err    error
}

// ProgramSyncBridge sends sync requests into a running Bubble Tea program.
type ProgramSyncBridge struct {
	program *tea.Program
}

// Bind attaches the running program used to deliver sync requests.
func (b *ProgramSyncBridge) Bind(program *tea.Program) {
	b.program = program
}

// Request triggers a sync cycle in the TUI.
func (b *ProgramSyncBridge) Request() {
	if b.program != nil {
		b.program.Send(SyncRequestMsg{})
	}
}

func (m Model) shouldAutoSync() bool {
	return !m.state.Config.OfflineMode && m.state.Session.Authenticated && m.syncRunner != nil
}

func (m Model) scheduleSyncTick() tea.Cmd {
	if !m.shouldAutoSync() {
		return nil
	}
	return tea.Tick(syncInterval, func(time.Time) tea.Msg {
		return syncTickMsg{}
	})
}

func (m Model) startSync() (tea.Model, tea.Cmd) {
	if m.syncRunner == nil || !m.shouldAutoSync() || m.syncSyncing {
		return m, nil
	}
	m.syncSyncing = true
	m.syncStatus = SyncStatusSyncing
	return m, m.syncCmd()
}

func (m Model) syncCmd() tea.Cmd {
	runner := m.syncRunner
	return func() tea.Msg {
		result, err := runner.Run(context.Background())
		return syncResultMsg{result: result, err: err}
	}
}

func (m Model) handleSyncResult(msg syncResultMsg) (tea.Model, tea.Cmd) {
	m.syncSyncing = false
	if msg.err != nil {
		m.syncStatus = SyncStatusError
		m.syncError = messages.UserMessage(msg.err)
		if m.syncRunner != nil {
			if count, err := m.syncRunner.PendingCount(context.Background()); err == nil {
				m.pendingCount = count
			}
		}
		return m, m.scheduleSyncTick()
	}

	m.syncError = ""
	m.pendingCount = msg.result.PendingCount
	m.lastSyncAt = time.Now()
	if m.pendingCount > 0 {
		m.syncStatus = SyncStatusPending
	} else {
		m.syncStatus = SyncStatusSynced
	}
	m.refreshMovies()
	m.persistState()
	return m, m.scheduleSyncTick()
}

func (m *Model) refreshPendingCount() {
	if m.syncRunner == nil {
		return
	}
	count, err := m.syncRunner.PendingCount(context.Background())
	if err == nil {
		m.pendingCount = count
		if count > 0 && m.syncStatus != SyncStatusSyncing && m.syncStatus != SyncStatusError {
			m.syncStatus = SyncStatusPending
		}
	}
}

func (m *Model) currentUserID() string {
	if m.resolveUserID != nil {
		return m.resolveUserID()
	}
	return effectiveUserID(m.state)
}
