package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	route      Route
	previous   Route
	state      AppState
	width      int
	height     int
	menu       list.Model
	movies     list.Model
	themeInput textinput.Model
	emailInput textinput.Model
	notes      textarea.Model
	message    string
}

func New() Model {
	menu := list.New(mainMenuItems(), list.NewDefaultDelegate(), 0, 0)
	menu.Title = "Menu principal"
	menu.SetShowStatusBar(false)
	menu.SetFilteringEnabled(false)

	movies := list.New(movieListItems(), list.NewDefaultDelegate(), 0, 0)
	movies.Title = "Films"
	movies.SetShowStatusBar(false)
	movies.SetFilteringEnabled(false)

	themeInput := textinput.New()
	themeInput.Placeholder = "midnight"
	themeInput.SetValue("midnight")
	themeInput.CharLimit = 32

	emailInput := textinput.New()
	emailInput.Placeholder = "vous@example.com"
	emailInput.CharLimit = 80

	notes := textarea.New()
	notes.Placeholder = "Notes de navigation pour la phase TUI..."
	notes.SetValue("Phase 2 pose la coquille. Les données réelles arrivent avec la phase 3.")
	notes.SetWidth(64)
	notes.SetHeight(5)

	return Model{
		route:      RouteSplash,
		state:      defaultState(),
		menu:       menu,
		movies:     movies,
		themeInput: themeInput,
		emailInput: emailInput,
		notes:      notes,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateActiveBubble(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		if m.route != RouteMainMenu && m.route != RouteSplash {
			m.goTo(RouteMainMenu)
			return m, nil
		}
	case "?":
		m.goTo(RouteHelp)
		return m, nil
	case "h":
		if m.route != RouteSplash && m.route != RouteLogin && m.route != RouteSettings {
			m.goTo(RouteHelp)
			return m, nil
		}
	case "m":
		m.goTo(RouteMainMenu)
		return m, nil
	case "s":
		if m.route != RouteSplash && m.route != RouteSettings && m.route != RouteLogin {
			m.goTo(RouteSettings)
			return m, nil
		}
	case "l":
		if m.route != RouteSplash && m.route != RouteLogin && m.route != RouteSettings {
			m.goTo(RouteLogin)
			return m, nil
		}
	}

	switch m.route {
	case RouteSplash:
		return m.updateSplash(msg)
	case RouteMainMenu:
		return m.updateMainMenu(msg)
	case RouteMovieList:
		return m.updateMovieList(msg)
	case RouteSettings:
		return m.updateSettings(msg)
	case RouteLogin:
		return m.updateLogin(msg)
	case RouteMovieDetail:
		return m.updateMovieDetail(msg)
	default:
		return m.updateActiveBubble(msg)
	}
}

func (m Model) updateSplash(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		m.goTo(RouteMainMenu)
	}
	return m, nil
}

func (m Model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		if item, ok := m.menu.SelectedItem().(menuItem); ok {
			m.goTo(item.route)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m Model) updateMovieList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		m.goTo(RouteMovieDetail)
		return m, nil
	}

	var cmd tea.Cmd
	m.movies, cmd = m.movies.Update(msg)
	return m, cmd
}

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(m.themeInput.Value())
		if value == "" {
			m.message = "Le thème ne peut pas être vide."
			return m, nil
		}
		m.state.Config.Theme = value
		m.message = fmt.Sprintf("Thème actif : %s", value)
		return m, nil
	}

	var cmd tea.Cmd
	m.themeInput, cmd = m.themeInput.Update(msg)
	return m, cmd
}

func (m Model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		email := strings.TrimSpace(m.emailInput.Value())
		if email == "" {
			m.message = "Saisis un email pour simuler la connexion."
			return m, nil
		}
		m.state.User.Email = email
		m.message = "Session locale prête pour " + email
		return m, nil
	}

	var cmd tea.Cmd
	m.emailInput, cmd = m.emailInput.Update(msg)
	return m, cmd
}

func (m Model) updateMovieDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.notes, cmd = m.notes.Update(msg)
	return m, cmd
}

func (m Model) updateActiveBubble(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.route {
	case RouteMainMenu:
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd
	case RouteMovieList:
		var cmd tea.Cmd
		m.movies, cmd = m.movies.Update(msg)
		return m, cmd
	case RouteSettings:
		var cmd tea.Cmd
		m.themeInput, cmd = m.themeInput.Update(msg)
		return m, cmd
	case RouteLogin:
		var cmd tea.Cmd
		m.emailInput, cmd = m.emailInput.Update(msg)
		return m, cmd
	case RouteMovieDetail:
		var cmd tea.Cmd
		m.notes, cmd = m.notes.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) goTo(route Route) {
	m.previous = m.route
	m.route = route
	m.message = ""
	m.themeInput.Blur()
	m.emailInput.Blur()
	m.notes.Blur()

	switch route {
	case RouteSettings:
		m.themeInput.Focus()
	case RouteLogin:
		m.emailInput.Focus()
	case RouteMovieDetail:
		m.notes.Focus()
	}
}

func (m *Model) resizeLists() {
	listWidth := m.width - 8
	if listWidth < 30 {
		listWidth = 30
	}

	listHeight := m.height - 9
	if listHeight < 8 {
		listHeight = 8
	}

	m.menu.SetSize(listWidth, listHeight)
	m.movies.SetSize(listWidth, listHeight)
	m.notes.SetWidth(listWidth)
}

func mainMenuItems() []list.Item {
	return []list.Item{
		menuItem{"Films", "Parcourir la liste locale", RouteMovieList},
		menuItem{"Statistiques", "Voir les indicateurs de suivi", RouteStats},
		menuItem{"Paramètres", "Changer le thème et les préférences", RouteSettings},
		menuItem{"Connexion", "Préparer l'authentification serveur", RouteLogin},
		menuItem{"Aide", "Afficher les raccourcis", RouteHelp},
	}
}

func movieListItems() []list.Item {
	return []list.Item{
		movieItem{"Arrival", "Exemple phase 2 - détail avec Entrée"},
		movieItem{"Heat", "Exemple phase 2 - non connecté à la DB"},
		movieItem{"The Matrix", "Exemple phase 2 - navigation seulement"},
	}
}
