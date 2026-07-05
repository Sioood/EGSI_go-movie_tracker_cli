package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyboardNavigationBetweenScreens(t *testing.T) {
	model := New()

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMainMenu)

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMovieList)

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMovieDetail)

	model = press(t, model, "esc")
	assertRoute(t, model, RouteMainMenu)

	model.goTo(RouteStats)
	assertRoute(t, model, RouteStats)

	model = press(t, model, "s")
	assertRoute(t, model, RouteSettings)

	model = press(t, model, "esc")
	assertRoute(t, model, RouteMainMenu)

	model = press(t, model, "l")
	assertRoute(t, model, RouteLogin)

	model = press(t, model, "esc")
	assertRoute(t, model, RouteMainMenu)

	model = press(t, model, "?")
	assertRoute(t, model, RouteHelp)
}

func press(t *testing.T, model Model, key string) Model {
	t.Helper()

	updated, _ := model.Update(keyMsg(key))
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected tui.Model, got %T", updated)
	}
	return next
}

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func assertRoute(t *testing.T, model Model, route Route) {
	t.Helper()

	if model.route != route {
		t.Fatalf("expected route %s, got %s", route, model.route)
	}
}
