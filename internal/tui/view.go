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

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)
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
	user := connectionStatus(m.state)
	right := subtleStyle.Render(messages.ThemeHeader(m.state.Config.Theme, user))
	line := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render(messages.UI.AppName+" v"+version.Version), "  ", right)
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
	}
	if syncLine != "" {
		return subtleStyle.Render(syncLine + "  |  " + help)
	}
	return subtleStyle.Render(help)
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
	case SyncStatusSynced:
		return messages.UI.SyncUpToDate
	default:
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
	default:
		return m.menu.View()
	}
}

func (m Model) splashView() string {
	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(messages.UI.SplashWelcome),
		"",
		messages.UI.SplashTagline,
		"",
		messages.UI.SplashEnter,
	}, "\n"))
}

func (m Model) movieListView() string {
	searchBlock := strings.Join([]string{
		label(messages.UI.SearchLabel, m.searchInput.Focused()),
		m.searchInput.View(),
		subtleStyle.Render(fmt.Sprintf(messages.UI.FilterSortHint, messages.FilterLabel(m.filter), messages.SortLabel(m.sort))),
	}, "\n")

	if len(m.movieRecords) == 0 {
		emptyMessage := messages.UI.EmptyMovies
		if strings.TrimSpace(m.searchInput.Value()) != "" || m.filter != domain.MovieFilterAll {
			emptyMessage = messages.UI.EmptySearch
		}
		return panelStyle.Render(strings.Join([]string{
			activeStyle.Render(messages.UI.MoviesTitle),
			"",
			searchBlock,
			"",
			emptyMessage,
			messages.UI.AddMovieHint,
			"",
			statusLine(m.messageKind, m.message),
		}, "\n"))
	}

	lines := []string{panelStyle.Render(searchBlock), m.movies.View()}
	if m.message != "" {
		lines = append(lines, "", statusLine(m.messageKind, m.message))
	}
	return strings.Join(lines, "\n")
}

func (m Model) movieFormView() string {
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.MovieFormHint
		kind = messages.KindInfo
	}

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(messages.UI.AddMovieTitle),
		"",
		label(messages.UI.TitleLabel, m.formFocus == 0),
		m.titleInput.View(),
		"",
		label(messages.UI.YearLabel, m.formFocus == 1),
		m.yearInput.View(),
		"",
		statusLine(kind, message),
	}, "\n"))
}

func (m Model) movieDetailView() string {
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

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(title),
		subtleStyle.Render("Vu : " + watched),
		"",
		label(messages.UI.RatingLabel, m.detailFocus == 0),
		m.ratingInput.View(),
		"",
		label(messages.UI.WatchedAtLabel, m.detailFocus == 1),
		m.watchedAtInput.View(),
		"",
		label(messages.UI.ReviewLabel, m.detailFocus == 2),
		m.reviewInput.View(),
		"",
		statusLine(kind, message),
	}, "\n"))
}

func (m Model) statsView() string {
	s := m.stats
	lines := []string{
		activeStyle.Render(messages.UI.StatsTitle),
		"",
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalMovies, s.TotalMovies),
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalWatched, s.TotalWatched),
		fmt.Sprintf("%-22s %d", messages.UI.StatsTotalRated, s.TotalRated),
	}

	if s.TotalRated > 0 {
		lines = append(lines, fmt.Sprintf("%-22s "+messages.UI.StatsRatingFmt, messages.UI.StatsAverageRating, s.AverageRating))
	} else {
		lines = append(lines, fmt.Sprintf("%-22s %s", messages.UI.StatsAverageRating, subtleStyle.Render("—")))
	}

	if len(s.BestMovies) > 0 {
		lines = append(lines, "", labelStyle.Render(messages.UI.StatsBestMovies))
		for _, mr := range s.BestMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s "+messages.UI.StatsRatingFmt, title, mr.Rating))
		}
	}

	if len(s.WorstMovies) > 0 {
		lines = append(lines, "", labelStyle.Render(messages.UI.StatsWorstMovies))
		for _, mr := range s.WorstMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s "+messages.UI.StatsRatingFmt, title, mr.Rating))
		}
	}

	if len(s.ByMonth) > 0 {
		lines = append(lines, "", labelStyle.Render(messages.UI.StatsByMonth))
		lines = append(lines, asciiHistogram(s.ByMonth))
	}

	if s.TotalMovies == 0 {
		lines = append(lines, "", subtleStyle.Render(messages.UI.StatsEmptyHint))
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) settingsView() string {
	message := m.message
	kind := m.messageKind
	if message == "" {
		message = messages.UI.SettingsHint
		kind = messages.KindInfo
	}

	lines := []string{
		activeStyle.Render(messages.UI.SettingsTitle),
		"",
		label(messages.UI.ThemeLabel, m.settingsFocus == 0),
		m.themeInput.View(),
		"",
		label(messages.UI.ServerURLLabel, m.settingsFocus == 1),
		m.serverURLInput.View(),
		"",
		subtleStyle.Render(fmt.Sprintf(messages.UI.OfflineToggleHint, messages.OfflineModeLabel(m.state.Config.OfflineMode))),
	}
	if m.state.Session.Authenticated {
		lines = append(lines, subtleStyle.Render(fmt.Sprintf(messages.UI.ConnectedHintFmt, m.state.Session.Email)))
	}
	lines = append(lines, "", statusLine(kind, message))
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) loginView() string {
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

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(messages.UI.LoginTitle),
		"",
		label(messages.UI.EmailLabel, m.loginFocus == 0),
		m.emailInput.View(),
		"",
		label(messages.UI.PasswordLabel, m.loginFocus == 1),
		m.passwordInput.View(),
		"",
		subtleStyle.Render(messages.UI.LoginNoAccount),
		"",
		statusLine(kind, message),
	}, "\n"))
}

func (m Model) registerView() string {
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

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(messages.UI.RegisterTitle),
		"",
		label(messages.UI.EmailLabel, m.registerFocus == 0),
		m.emailInput.View(),
		"",
		label(messages.UI.PasswordLabel, m.registerFocus == 1),
		m.passwordInput.View(),
		"",
		label(messages.UI.ConfirmLabel, m.registerFocus == 2),
		m.confirmPasswordInput.View(),
		"",
		statusLine(kind, message),
	}, "\n"))
}

func asciiHistogram(buckets []domain.MonthBucket) string {
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
		bar := activeStyle.Render(strings.Repeat("█", barLen))
		lines = append(lines, fmt.Sprintf("  %-9s %s %d", lbl, bar, b.Count))
	}
	return strings.Join(lines, "\n")
}

func label(text string, active bool) string {
	if active {
		return activeStyle.Render(text)
	}
	return labelStyle.Render(text)
}

func statusLine(kind messages.Kind, message string) string {
	if message == "" {
		return ""
	}
	switch kind {
	case messages.KindError:
		return errorStyle.Render(message)
	case messages.KindSuccess:
		return activeStyle.Render(message)
	default:
		return subtleStyle.Render(message)
	}
}
