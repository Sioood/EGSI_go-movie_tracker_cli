package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

type conflictItem struct {
	conflict domain.SyncConflict
	title    string
}

func (i conflictItem) Title() string       { return i.title }
func (i conflictItem) Description() string { return messages.UI.ConflictListHint }
func (i conflictItem) FilterValue() string { return i.title }

func (m *Model) refreshConflicts() {
	if m.syncRunner == nil {
		m.conflicts.SetItems(nil)
		m.conflictRecords = nil
		return
	}
	records, err := m.syncRunner.ListConflicts(m.appContext())
	if err != nil {
		m.setMessage(messages.KindError, fmt.Sprintf(messages.UI.LoadFailedFmt, messages.UserMessage(err)))
		return
	}
	m.conflictRecords = records
	items := make([]list.Item, 0, len(records))
	for _, conflict := range records {
		items = append(items, conflictItem{
			conflict: conflict,
			title:    m.conflictTitle(conflict),
		})
	}
	m.conflicts.SetItems(items)
	if len(items) > 0 {
		m.conflicts.Select(0)
		m.selectedConflict = records[0]
		m.conflictChoice = domain.ConflictChoiceLocal
	}
}

func (m Model) conflictTitle(conflict domain.SyncConflict) string {
	switch conflict.EntityType {
	case domain.SyncEntityMovie:
		var movie domain.Movie
		if err := json.Unmarshal([]byte(conflict.LocalJSON), &movie); err == nil && movie.Title != "" {
			return fmt.Sprintf(messages.UI.ConflictMovieFmt, movie.Title)
		}
	case domain.SyncEntityWatchEntry:
		var entry domain.WatchEntry
		if err := json.Unmarshal([]byte(conflict.LocalJSON), &entry); err == nil {
			return fmt.Sprintf(messages.UI.ConflictWatchFmt, entry.MovieID)
		}
	}
	return fmt.Sprintf(messages.UI.ConflictGenericFmt, conflict.EntityType, conflict.EntityID)
}

func (m Model) updateConflicts(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab":
		if m.conflictChoice == domain.ConflictChoiceLocal {
			m.conflictChoice = domain.ConflictChoiceRemote
		} else {
			m.conflictChoice = domain.ConflictChoiceLocal
		}
		return m, nil
	case "enter":
		if m.syncRunner == nil || m.selectedConflict.ID == "" {
			return m, nil
		}
		if err := m.syncRunner.ResolveConflict(m.appContext(), m.selectedConflict.ID, m.conflictChoice); err != nil {
			m.setError(err)
			return m, nil
		}
		m.setMessage(messages.KindSuccess, messages.UI.ConflictResolved)
		m.refreshConflicts()
		m.refreshConflictCount()
		m.refreshMovies()
		if m.conflictCount == 0 {
			m.goTo(RouteMainMenu)
			return m, nil
		}
		return m, m.syncCmd()
	case "up", "down", "k", "j":
		var cmd tea.Cmd
		m.conflicts, cmd = m.conflicts.Update(msg)
		if item, ok := m.conflicts.SelectedItem().(conflictItem); ok {
			m.selectedConflict = item.conflict
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) conflictsView() string {
	s := m.styles
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.ConflictScreenHint
		kind = messages.KindInfo
	}

	lines := []string{
		s.Active.Render(messages.UI.ConflictTitle),
		"",
	}
	if len(m.conflictRecords) == 0 {
		lines = append(lines, messages.UI.ConflictEmpty)
	} else {
		lines = append(lines, m.conflicts.View(), "", m.conflictPreview())
	}
	lines = append(lines, "", m.statusLine(kind, message))
	return s.Panel.Render(strings.Join(lines, "\n"))
}

func (m Model) conflictPreview() string {
	if m.selectedConflict.ID == "" {
		return ""
	}
	localLabel := m.deviceLabel(m.selectedConflict.LocalDeviceID, messages.UI.ConflictLocalLabel)
	remoteLabel := m.deviceLabel(m.selectedConflict.RemoteDeviceID, messages.UI.ConflictRemoteLabel)
	choice := messages.UI.ConflictLocalLabel
	if m.conflictChoice == domain.ConflictChoiceRemote {
		choice = messages.UI.ConflictRemoteLabel
	}
	return strings.Join([]string{
		fmt.Sprintf(messages.UI.ConflictChoiceFmt, choice),
		fmt.Sprintf(messages.UI.ConflictLocalPreviewFmt, localLabel, summarizeConflictJSON(m.selectedConflict.LocalJSON)),
		fmt.Sprintf(messages.UI.ConflictRemotePreviewFmt, remoteLabel, summarizeConflictJSON(m.selectedConflict.RemoteJSON)),
	}, "\n")
}

func (m Model) deviceLabel(deviceID, fallback string) string {
	if deviceID == "" {
		return fallback
	}
	if m.syncRunner != nil {
		if name, err := m.syncRunner.GetDeviceName(m.appContext(), deviceID); err == nil && name != "" {
			return name
		}
	}
	if deviceID == m.state.Config.DeviceID && m.state.Config.DeviceName != "" {
		return m.state.Config.DeviceName
	}
	return deviceID
}

func summarizeConflictJSON(raw string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{"title", "year", "watched", "rating", "review"} {
		if value, ok := payload[key]; ok && value != nil && value != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", key, value))
		}
	}
	if len(parts) == 0 {
		return messages.UI.ConflictNoPreview
	}
	return strings.Join(parts, ", ")
}

func (m *Model) refreshConflictCount() {
	if m.syncRunner == nil {
		m.conflictCount = 0
		return
	}
	count, err := m.syncRunner.ConflictCount(m.appContext())
	if err == nil {
		m.conflictCount = count
		if count > 0 && m.syncStatus != SyncStatusSyncing && m.syncStatus != SyncStatusError {
			m.syncStatus = SyncStatusConflicts
		}
	}
}

func (m Model) openConflicts() (tea.Model, tea.Cmd) {
	if m.conflictCount == 0 {
		m.setMessage(messages.KindInfo, messages.UI.ConflictEmpty)
		return m, nil
	}
	m.refreshConflicts()
	m.goTo(RouteSyncConflicts)
	return m, nil
}
