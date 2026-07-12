package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

// BackupSnapshot is the local representation of a remote config/state backup.
type BackupSnapshot struct {
	Config config.Config
	State  config.State
}

// BackupClient imports and exports config/state snapshots from the server.
type BackupClient interface {
	ExportSnapshot(ctx context.Context, accessToken string) (BackupSnapshot, error)
	ImportSnapshot(ctx context.Context, accessToken string, snapshot BackupSnapshot) error
}

type backupResultMsg struct {
	snapshot BackupSnapshot
	action   string
	err      error
}

func (m Model) exportBackupCmd(accessToken string) tea.Cmd {
	client := m.backup
	parent := m.appContext()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, backupRequestTimeout)
		defer cancel()

		snapshot, err := client.ExportSnapshot(ctx, accessToken)
		return backupResultMsg{snapshot: snapshot, action: "import", err: err}
	}
}

func (m Model) importBackupCmd(accessToken string, snapshot BackupSnapshot) tea.Cmd {
	client := m.backup
	parent := m.appContext()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, backupRequestTimeout)
		defer cancel()

		err := client.ImportSnapshot(ctx, accessToken, snapshot)
		return backupResultMsg{action: "export", err: err}
	}
}

func (m Model) currentBackupSnapshot() BackupSnapshot {
	lastSync := ""
	if !m.lastSyncAt.IsZero() {
		lastSync = m.lastSyncAt.UTC().Format(time.RFC3339)
	}
	return BackupSnapshot{
		Config: config.Config{
			Theme:       m.state.Config.Theme,
			ServerURL:   m.state.Config.ServerURL,
			OfflineMode: m.state.Config.OfflineMode,
		},
		State: config.State{
			LastRoute:  string(m.route),
			Filter:     string(m.filter),
			Sort:       string(m.sort),
			LastSyncAt: lastSync,
		},
	}
}

func (m *Model) applyBackupSnapshot(snapshot BackupSnapshot) {
	m.state.Config.Theme = NormalizeTheme(snapshot.Config.Theme)
	m.state.Config.ServerURL = snapshot.Config.ServerURL
	m.state.Config.OfflineMode = snapshot.Config.OfflineMode
	m.applyTheme()

	m.serverURLInput.SetValue(m.state.Config.ServerURL)

	if snapshot.State.Filter != "" {
		m.filter = domainMovieFilter(snapshot.State.Filter)
	}
	if snapshot.State.Sort != "" {
		m.sort = domainMovieSort(snapshot.State.Sort)
	}
	if snapshot.State.LastSyncAt != "" {
		if parsed, err := time.Parse(time.RFC3339, snapshot.State.LastSyncAt); err == nil {
			m.lastSyncAt = parsed
		}
	}
}

func domainMovieFilter(value string) domain.MovieFilter {
	switch domain.MovieFilter(value) {
	case domain.MovieFilterWatched, domain.MovieFilterUnwatched, domain.MovieFilterRated, domain.MovieFilterUnrated:
		return domain.MovieFilter(value)
	default:
		return domain.MovieFilterAll
	}
}

func domainMovieSort(value string) domain.MovieSort {
	switch domain.MovieSort(value) {
	case domain.MovieSortDate, domain.MovieSortRating:
		return domain.MovieSort(value)
	default:
		return domain.MovieSortTitle
	}
}

func (m Model) handleBackupResult(msg backupResultMsg) (tea.Model, tea.Cmd) {
	m.backupLoading = false
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}

	switch msg.action {
	case "import":
		m.applyBackupSnapshot(msg.snapshot)
		if m.saveConfig != nil {
			if err := m.saveConfig(m.state.Config); err != nil {
				m.setMessage(messages.KindError, fmt.Sprintf(messages.UI.SaveFailedFmt, messages.UserMessage(err)))
				return m, nil
			}
		}
		m.persistState()
		m.setMessage(messages.KindSuccess, messages.UI.BackupImportOK)
	case "export":
		m.setMessage(messages.KindSuccess, messages.UI.BackupExportOK)
	}
	return m, nil
}
