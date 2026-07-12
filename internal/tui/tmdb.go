package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/tmdb"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

const (
	movieFormModeManual = "manual"
	movieFormModeSearch = "search"
)

// TMDBSearcher performs remote TMDB lookups for the add-movie form.
type TMDBSearcher interface {
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error)
}

type tmdbSearchResultMsg struct {
	results []tmdb.SearchResult
	err     error
}

type tmdbResultItem struct {
	result tmdb.SearchResult
}

func (i tmdbResultItem) Title() string {
	if i.result.Year > 0 {
		return fmt.Sprintf("%s (%d)", i.result.Title, i.result.Year)
	}
	return i.result.Title
}

func (i tmdbResultItem) Description() string {
	overview := strings.TrimSpace(i.result.Overview)
	if overview == "" {
		return "Aucun résumé"
	}
	if len(overview) > 80 {
		return overview[:77] + "..."
	}
	return overview
}

func (i tmdbResultItem) FilterValue() string {
	return i.Title()
}

func (m *Model) enterTMDBSearch() (tea.Model, tea.Cmd) {
	if m.tmdbSearch == nil {
		m.setMessage(messages.KindError, messages.UI.TMDBUnavailable)
		return m, nil
	}
	m.movieFormMode = movieFormModeSearch
	m.tmdbResultsData = nil
	m.tmdbResults.SetItems(nil)
	m.tmdbQueryInput.SetValue(m.titleInput.Value())
	m.tmdbSearching = false
	m.clearFocus()
	m.tmdbQueryInput.Focus()
	m.setMessage(messages.KindInfo, messages.UI.TMDBSearchHint)
	return m, textinput.Blink
}

func (m Model) tmdbSearchCmd(query string) tea.Cmd {
	searcher := m.tmdbSearch
	year, _ := parseOptionalYear(m.yearInput.Value())
	parent := m.appContext()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, tmdbRequestTimeout)
		defer cancel()
		results, err := searcher.SearchMovies(ctx, query, year)
		return tmdbSearchResultMsg{results: results, err: err}
	}
}

func (m Model) updateMovieFormSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.tmdbSearching {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.movieFormMode = movieFormModeManual
		m.tmdbResults.SetItems(nil)
		m.tmdbResultsData = nil
		m.focusMovieForm()
		m.setMessage(messages.KindInfo, messages.UI.MovieFormHint)
		return m, nil
	case "enter":
		if len(m.tmdbResultsData) > 0 {
			if item, ok := m.tmdbResults.SelectedItem().(tmdbResultItem); ok {
				return m.applyTMDBSelection(item.result)
			}
		}
		query := strings.TrimSpace(m.tmdbQueryInput.Value())
		if len(query) < 2 {
			m.setMessage(messages.KindError, messages.UI.TMDBQueryTooShort)
			return m, nil
		}
		m.tmdbSearching = true
		m.setMessage(messages.KindInfo, messages.UI.TMDBSearching)
		return m, m.tmdbSearchCmd(query)
	case "up", "down", "k", "j":
		if len(m.tmdbResultsData) == 0 {
			var cmd tea.Cmd
			m.tmdbQueryInput, cmd = m.tmdbQueryInput.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.tmdbResults, cmd = m.tmdbResults.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.tmdbQueryInput, cmd = m.tmdbQueryInput.Update(msg)
	return m, cmd
}

func (m Model) applyTMDBSelection(result tmdb.SearchResult) (tea.Model, tea.Cmd) {
	m.titleInput.SetValue(result.Title)
	if result.Year > 0 {
		m.yearInput.SetValue(strconv.Itoa(result.Year))
	} else {
		m.yearInput.SetValue("")
	}
	m.selectedExternalID = fmt.Sprintf("tmdb:%d", result.ID)
	m.movieFormMode = movieFormModeManual
	m.tmdbResults.SetItems(nil)
	m.tmdbResultsData = nil
	m.formFocus = 0
	m.focusMovieForm()
	m.setMessage(messages.KindSuccess, fmt.Sprintf(messages.UI.TMDBSelectedFmt, result.Title))
	return m, nil
}

func (m *Model) handleTMDBSearchResult(msg tmdbSearchResultMsg) (tea.Model, tea.Cmd) {
	m.tmdbSearching = false
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}
	if len(msg.results) == 0 {
		m.tmdbResults.SetItems(nil)
		m.tmdbResultsData = nil
		m.setMessage(messages.KindInfo, messages.UI.TMDBNoResults)
		return m, nil
	}

	m.tmdbResultsData = msg.results
	items := make([]list.Item, 0, len(msg.results))
	for _, result := range msg.results {
		items = append(items, tmdbResultItem{result: result})
	}
	m.tmdbResults.SetItems(items)
	if len(items) > 0 {
		m.tmdbResults.Select(0)
	}
	m.setMessage(messages.KindInfo, messages.UI.TMDBResultsHint)
	return m, nil
}
