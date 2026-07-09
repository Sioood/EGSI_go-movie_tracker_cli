package tui

import (
	"time"

	"github.com/movietracker/movie-tracker/internal/config"
)

func (m *Model) applyTheme() {
	m.state.Config.Theme = NormalizeTheme(m.state.Config.Theme)
	m.styles = BuildThemeStyles(m.state.Config.Theme)
}

func (m *Model) persistState() {
	if m.saveState == nil {
		return
	}
	lastSync := ""
	if !m.lastSyncAt.IsZero() {
		lastSync = m.lastSyncAt.UTC().Format(time.RFC3339)
	}
	_ = m.saveState(config.State{
		LastRoute:  string(m.route),
		Filter:     string(m.filter),
		Sort:       string(m.sort),
		LastSyncAt: lastSync,
	})
}

// ParseRoute returns a valid route from persisted state.
func ParseRoute(value string) Route {
	switch Route(value) {
	case RouteSplash, RouteMainMenu, RouteMovieList, RouteMovieForm,
		RouteMovieDetail, RouteStats, RouteSettings, RouteLogin, RouteRegister, RouteHelp, RouteSyncConflicts:
		return Route(value)
	default:
		return RouteSplash
	}
}
