package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	user := "hors ligne"
	if m.state.User.Email != "" {
		user = m.state.User.Email
	}

	right := subtleStyle.Render(fmt.Sprintf("theme %s | %s", m.state.Config.Theme, user))
	line := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("MovieTracker"), "  ", right)
	return line
}

func (m Model) footerView() string {
	help := "↑/↓ naviguer • entrée sélectionner • esc menu • ? aide • q quitter"
	if m.route == RouteSplash {
		help = "entrée commencer • q quitter"
	}
	if m.route == RouteSettings || m.route == RouteLogin || m.route == RouteMovieDetail {
		help = "saisir du texte • entrée valider • esc menu • q quitter"
	}
	return subtleStyle.Render(help)
}

func (m Model) bodyView() string {
	switch m.route {
	case RouteSplash:
		return m.splashView()
	case RouteMainMenu:
		return m.menu.View()
	case RouteMovieList:
		return m.movies.View()
	case RouteMovieDetail:
		return m.movieDetailView()
	case RouteStats:
		return m.statsView()
	case RouteSettings:
		return m.settingsView()
	case RouteLogin:
		return m.loginView()
	case RouteHelp:
		return m.helpView()
	default:
		return m.menu.View()
	}
}

func (m Model) splashView() string {
	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Bienvenue dans MovieTracker"),
		"",
		"Une TUI pour suivre les films vus, les notes et les critiques.",
		"",
		"Appuie sur Entrée pour ouvrir le menu.",
	}, "\n"))
}

func (m Model) movieDetailView() string {
	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Détail film"),
		"",
		"Cet écran recevra les données SQLite en phase 3.",
		"Le composant textarea est déjà présent pour préparer les critiques.",
		"",
		m.notes.View(),
	}, "\n"))
}

func (m Model) statsView() string {
	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Statistiques"),
		"",
		"Films suivis        3 exemples",
		"Films vus           à connecter en phase 5",
		"Note moyenne        à connecter en phase 5",
		"",
		"Depuis cet écran : m menu, s paramètres, l connexion, ? aide.",
	}, "\n"))
}

func (m Model) settingsView() string {
	message := m.message
	if message == "" {
		message = "Modifie le thème puis valide avec Entrée."
	}

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Paramètres"),
		"",
		"Thème",
		m.themeInput.View(),
		"",
		subtleStyle.Render("Serveur : " + m.state.Config.ServerURL),
		subtleStyle.Render(fmt.Sprintf("Mode hors ligne : %t", m.state.Config.OfflineMode)),
		"",
		statusLine(message),
	}, "\n"))
}

func (m Model) loginView() string {
	message := m.message
	if message == "" {
		message = "Saisis un email pour préparer l'écran d'authentification."
	}

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Connexion"),
		"",
		"Email",
		m.emailInput.View(),
		"",
		statusLine(message),
	}, "\n"))
}

func (m Model) helpView() string {
	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Aide"),
		"",
		"Entrée      sélectionner ou valider",
		"↑/↓         naviguer dans les listes",
		"Esc ou m    revenir au menu",
		"s           paramètres",
		"l           connexion",
		"? ou h      aide",
		"q           quitter",
	}, "\n"))
}

func statusLine(message string) string {
	if strings.Contains(strings.ToLower(message), "ne peut pas") || strings.Contains(strings.ToLower(message), "saisis") {
		return errorStyle.Render(message)
	}
	return activeStyle.Render(message)
}
