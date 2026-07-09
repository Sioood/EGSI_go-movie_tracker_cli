package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
	"github.com/movietracker/movie-tracker/internal/version"
)

func (m Model) View() string {
	body := m.bodyView()
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		"",
		body,
		"",
		m.footerView(),
	)

	if m.width <= 0 {
		return content
	}

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Left, content)
}

func (m Model) headerView() string {
	s := m.styles
	user := connectionStatus(m.state)
	right := s.Subtle.Render(messages.ThemeHeader(m.state.Config.Theme, user))
	line := lipgloss.JoinHorizontal(lipgloss.Top, s.Title.Render(messages.UI.AppName+" v"+version.Version), "  ", right)
	return line
}

func connectionStatus(state AppState) string {
	if state.Config.OfflineMode {
		return messages.UI.StatusOffline
	}
	if state.Session.Authenticated && state.Session.Email != "" {
		return state.Session.Email + messages.UI.StatusOnlineSuffix
	}
	return messages.UI.StatusDisconnected
}

func (m Model) footerView() string {
	s := m.styles
	syncLine := m.syncFooterLine()
	help := messages.UI.FooterDefault
	switch m.route {
	case RouteSplash:
		help = messages.UI.FooterSplash
	case RouteMovieForm:
		help = messages.UI.FooterMovieForm
	case RouteMovieDetail:
		help = messages.UI.FooterMovieDetail
	case RouteStats:
		help = messages.UI.FooterStats
	case RouteSettings:
		help = messages.UI.FooterSettings
	case RouteLogin:
		help = messages.UI.FooterLogin
	case RouteRegister:
		help = messages.UI.FooterRegister
	case RouteSyncConflicts:
		help = messages.UI.FooterConflicts
	}
	if syncLine != "" {
		return s.Subtle.Render(syncLine + "  |  " + help)
	}
	return s.Subtle.Render(help)
}

func (m Model) syncFooterLine() string {
	if m.state.Config.OfflineMode {
		return messages.UI.SyncOffline
	}
	switch m.syncStatus {
	case SyncStatusSyncing:
		return messages.UI.SyncSyncing
	case SyncStatusError:
		detail := strings.TrimSpace(m.syncError)
		if detail == "" {
			return messages.UI.SyncError
		}
		return messages.UI.SyncError + " : " + truncate(detail, 40)
	case SyncStatusPending:
		return messages.SyncPendingLine(m.pendingCount)
	case SyncStatusConflicts:
		return fmt.Sprintf(messages.UI.SyncConflictsFmt, m.conflictCount)
	case SyncStatusSynced:
		return messages.UI.SyncUpToDate
	default:
		if m.conflictCount > 0 {
			return fmt.Sprintf(messages.UI.SyncConflictsFmt, m.conflictCount)
		}
		if m.pendingCount > 0 {
			return messages.SyncPendingLine(m.pendingCount)
		}
		return messages.UI.SyncReady
	}
}

func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	if max <= 3 {
		return text[:max]
	}
	return text[:max-3] + "..."
}

func (m Model) bodyView() string {
	switch m.route {
	case RouteSplash:
		return m.splashView()
	case RouteMainMenu:
		return m.menu.View()
	case RouteMovieList:
		return m.movieListView()
	case RouteMovieForm:
		return m.movieFormView()
	case RouteMovieDetail:
		return m.movieDetailView()
	case RouteStats:
		return m.statsView()
	case RouteSettings:
		return m.settingsView()
	case RouteLogin:
		return m.loginView()
	case RouteRegister:
		return m.registerView()
	case RouteHelp:
		return m.helpView()
	case RouteSyncConflicts:
		return m.conflictsView()
	default:
		return m.menu.View()
	}
}

func (m Model) splashView() string {
	s := m.styles
	return s.Panel.Render(strings.Join([]string{
		s.Active.Render(messages.UI.SplashWelcome),
		"",
		messages.UI.SplashTagline,
		"",
		messages.UI.SplashEnter,
	}, "\n"))
}

func (m Model) movieListView() string {
	s := m.styles
	searchBlock := strings.Join([]string{
		m.label(messages.UI.SearchLabel, m.searchInput.Focused()),
		m.searchInput.View(),
		s.Subtle.Render(fmt.Sprintf(messages.UI.FilterSortHint, messages.FilterLabel(m.filter), messages.SortLabel(m.sort))),
	}, "\n")

	if len(m.movieRecords) == 0 {
		emptyMessage := messages.UI.EmptyMovies
		if strings.TrimSpace(m.searchInput.Value()) != "" || m.filter != domain.MovieFilterAll {
			emptyMessage = messages.UI.EmptySearch
		}
		return s.Panel.Render(strings.Join([]string{
			s.Active.Render(messages.UI.MoviesTitle),
			"",
			searchBlock,
			"",
			emptyMessage,
			messages.UI.AddMovieHint,
			"",
			m.statusLine(m.messageKind, m.message),
		}, "\n"))
	}

	lines := []string{s.Panel.Render(searchBlock), m.movies.View()}
	if m.message != "" {
		lines = append(lines, "", m.statusLine(m.messageKind, m.message))
	}
	return strings.Join(lines, "\n")
}

func (m Model) movieFormView() string {
	s := m.styles
	if m.movieFormMode == movieFormModeSearch {
		return m.movieFormSearchView()
	}

	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.MovieFormHint
		kind = messages.KindInfo
	}

	return s.Panel.Render(strings.Join([]string{
		s.Active.Render(messages.UI.AddMovieTitle),
		"",
		m.label(messages.UI.TitleLabel, m.formFocus == 0),
		m.titleInput.View(),
		"",
		m.label(messages.UI.YearLabel, m.formFocus == 1),
		m.yearInput.View(),
		"",
		m.statusLine(kind, message),
	}, "\n"))
}

func (m Model) movieFormSearchView() string {
	s := m.styles
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.TMDBSearchHint
		kind = messages.KindInfo
	}

	lines := []string{
		s.Active.Render(messages.UI.TMDBSearchTitle),
		"",
		m.label(messages.UI.TMDBSearchLabel, true),
		m.tmdbQueryInput.View(),
	}
	if len(m.tmdbResultsData) > 0 {
		lines = append(lines, "", m.tmdbResults.View())
	}
	lines = append(lines, "", m.statusLine(kind, message))
	return s.Panel.Render(strings.Join(lines, "\n"))
}

func (m Model) movieDetailView() string {
	s := m.styles
	title := messages.UI.MovieDetailTitle
	if m.selectedMovie.Title != "" {
		title = m.selectedMovie.Title
		if m.selectedMovie.Year > 0 {
			title = fmt.Sprintf("%s (%d)", title, m.selectedMovie.Year)
		}
	}

	watched := messages.UI.WatchedNo
	if m.selectedEntry.Watched {
		watched = messages.UI.WatchedYes
	}

	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.MovieDetailHint
		kind = messages.KindInfo
	}

	detailLines := []string{
		s.Active.Render(title),
		s.Subtle.Render("Vu : " + watched),
	}
	if m.selectedMovie.ExternalID != "" {
		detailLines = append(detailLines, s.Subtle.Render(fmt.Sprintf(messages.UI.ExternalIDFmt, m.selectedMovie.ExternalID)))
	}
	detailLines = append(detailLines,
		"",
		m.label(messages.UI.RatingLabel, m.detailFocus == 0),
		m.ratingInput.View(),
		"",
		m.label(messages.UI.WatchedAtLabel, m.detailFocus == 1),
		m.watchedAtInput.View(),
		"",
		m.label(messages.UI.ReviewLabel, m.detailFocus == 2),
		m.reviewInput.View(),
		"",
		m.statusLine(kind, message),
	)
	return s.Panel.Render(strings.Join(detailLines, "\n"))
}

func (m Model) statsView() string {
	s := m.styles
	stats := m.stats
	lines := []string{
		s.Active.Render(messages.UI.StatsTitle),
		"",
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalMovies, stats.TotalMovies),
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalWatched, stats.TotalWatched),
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalRated, stats.TotalRated),
	}

	if stats.TotalRated > 0 {
		lines = append(lines, fmt.Sprintf("%-22s "+messages.UI.StatsRatingFmt, messages.UI.StatsAverageRating, stats.AverageRating))
	} else {
		lines = append(lines, fmt.Sprintf("%-22s %s", messages.UI.StatsAverageRating, s.Subtle.Render("—")))
	}

	if len(stats.BestMovies) > 0 {
		lines = append(lines, "", s.Label.Render(messages.UI.StatsBestMovies))
		for _, mr := range stats.BestMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s "+messages.UI.StatsRatingFmt, title, mr.Rating))
		}
	}

	if len(stats.WorstMovies) > 0 {
		lines = append(lines, "", s.Label.Render(messages.UI.StatsWorstMovies))
		for _, mr := range stats.WorstMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s "+messages.UI.StatsRatingFmt, title, mr.Rating))
		}
	}

	if len(stats.ByMonth) > 0 {
		lines = append(lines, "", s.Label.Render(messages.UI.StatsByMonth))
		lines = append(lines, m.asciiHistogram(stats.ByMonth))
	}

	if stats.TotalMovies == 0 {
		lines = append(lines, "", s.Subtle.Render(messages.UI.StatsEmptyHint))
	}

	return s.Panel.Render(strings.Join(lines, "\n"))
}

func (m Model) settingsView() string {
	s := m.styles
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.SettingsHint
		kind = messages.KindInfo
	}

	themeValue := fmt.Sprintf("%s  (←/→)", NormalizeTheme(m.state.Config.Theme))
	lines := []string{
		s.Active.Render(messages.UI.SettingsTitle),
		"",
		m.label(messages.UI.ThemeLabel, m.settingsFocus == 0),
		themeValue,
		"",
		m.label(messages.UI.ServerURLLabel, m.settingsFocus == 1),
		m.serverURLInput.View(),
		"",
		s.Subtle.Render(fmt.Sprintf(messages.UI.OfflineToggleHint, messages.OfflineModeLabel(m.state.Config.OfflineMode))),
		s.Subtle.Render(messages.UI.BackupHint),
		s.Subtle.Render(messages.UI.ExportHint),
	}
	if m.state.Session.Authenticated {
		lines = append(lines, s.Subtle.Render(fmt.Sprintf(messages.UI.ConnectedHintFmt, m.state.Session.Email)))
	}
	lines = append(lines, "", m.statusLine(kind, message))
	return s.Panel.Render(strings.Join(lines, "\n"))
}

func (m Model) loginView() string {
	s := m.styles
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.LoginHint
		kind = messages.KindInfo
	}
	if m.authLoading {
		message = messages.UI.LoginLoading
		kind = messages.KindInfo
	}

	return s.Panel.Render(strings.Join([]string{
		s.Active.Render(messages.UI.LoginTitle),
		"",
		m.label(messages.UI.EmailLabel, m.loginFocus == 0),
		m.emailInput.View(),
		"",
		m.label(messages.UI.PasswordLabel, m.loginFocus == 1),
		m.passwordInput.View(),
		"",
		s.Subtle.Render(messages.UI.LoginNoAccount),
		"",
		m.statusLine(kind, message),
	}, "\n"))
}

func (m Model) registerView() string {
	s := m.styles
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.RegisterHint
		kind = messages.KindInfo
	}
	if m.authLoading {
		message = messages.UI.RegisterLoading
		kind = messages.KindInfo
	}

	return s.Panel.Render(strings.Join([]string{
		s.Active.Render(messages.UI.RegisterTitle),
		"",
		m.label(messages.UI.EmailLabel, m.registerFocus == 0),
		m.emailInput.View(),
		"",
		m.label(messages.UI.PasswordLabel, m.registerFocus == 1),
		m.passwordInput.View(),
		"",
		m.label(messages.UI.ConfirmLabel, m.registerFocus == 2),
		m.confirmPasswordInput.View(),
		"",
		m.statusLine(kind, message),
	}, "\n"))
}

func (m Model) asciiHistogram(buckets []domain.MonthBucket) string {
	if len(buckets) > 12 {
		buckets = buckets[len(buckets)-12:]
	}

	maxCount := 0
	for _, b := range buckets {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}

	const barWidth = 18
	monthNames := [12]string{"Jan", "Fév", "Mar", "Avr", "Mai", "Jun", "Jul", "Aoû", "Sep", "Oct", "Nov", "Déc"}

	var lines []string
	for _, b := range buckets {
		lbl := monthNames[b.Month-1] + " " + strconv.Itoa(b.Year)
		barLen := 0
		if maxCount > 0 {
			barLen = (b.Count * barWidth) / maxCount
		}
		bar := m.styles.Active.Render(strings.Repeat("█", barLen))
		lines = append(lines, fmt.Sprintf("  %-9s %s %d", lbl, bar, b.Count))
	}
	return strings.Join(lines, "\n")
}

func (m Model) label(text string, active bool) string {
	if active {
		return m.styles.Active.Render(text)
	}
	return m.styles.Label.Render(text)
}

func (m Model) statusLine(kind messages.Kind, message string) string {
	if message == "" {
		return ""
	}
	s := m.styles
	switch kind {
	case messages.KindError:
		return s.Error.Render(message)
	case messages.KindSuccess:
		return s.Active.Render(message)
	default:
		return s.Subtle.Render(message)
	}
}
