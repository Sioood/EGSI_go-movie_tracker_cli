package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/movietracker/movie-tracker/internal/domain"
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
	user := "hors ligne"
	if m.state.User.Email != "" {
		user = m.state.User.Email
	}

	right := subtleStyle.Render(fmt.Sprintf("theme %s | %s", m.state.Config.Theme, user))
	line := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("MovieTracker"), "  ", right)
	return line
}

func (m Model) footerView() string {
	help := "↑/↓ naviguer • entrée sélectionner • / chercher • f filtre • t tri • a ajouter • q quitter"
	switch m.route {
	case RouteSplash:
		help = "entrée commencer • q quitter"
	case RouteMovieForm:
		help = "tab changer de champ • entrée ajouter • esc liste • q quitter"
	case RouteMovieDetail:
		help = "tab champ suivant • w vu aujourd'hui • u non vu • entrée enregistrer • esc liste"
	case RouteStats:
		help = "m menu • s paramètres • l connexion • ? aide • q quitter"
	case RouteSettings, RouteLogin:
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

func (m Model) movieListView() string {
	searchBlock := strings.Join([]string{
		label("Recherche", m.searchInput.Focused()),
		m.searchInput.View(),
		subtleStyle.Render("Filtre : " + filterLabel(m.filter) + " | Tri : " + sortLabel(m.sort) + " | f filtre | t tri | c reset"),
	}, "\n")

	if len(m.movieRecords) == 0 {
		emptyMessage := "Aucun film enregistré pour l'instant."
		if strings.TrimSpace(m.searchInput.Value()) != "" || m.filter != "all" {
			emptyMessage = "Aucun film ne correspond à la recherche."
		}
		return panelStyle.Render(strings.Join([]string{
			activeStyle.Render("Films"),
			"",
			searchBlock,
			"",
			emptyMessage,
			"Appuie sur a pour ajouter un film.",
			"",
			statusLine(m.message),
		}, "\n"))
	}

	lines := []string{panelStyle.Render(searchBlock), m.movies.View()}
	if m.message != "" {
		lines = append(lines, "", statusLine(m.message))
	}
	return strings.Join(lines, "\n")
}

func (m Model) movieFormView() string {
	message := m.message
	if message == "" {
		message = "Titre obligatoire, année optionnelle."
	}

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render("Ajouter un film"),
		"",
		label("Titre", m.formFocus == 0),
		m.titleInput.View(),
		"",
		label("Année", m.formFocus == 1),
		m.yearInput.View(),
		"",
		statusLine(message),
	}, "\n"))
}

func (m Model) movieDetailView() string {
	title := "Détail film"
	if m.selectedMovie.Title != "" {
		title = m.selectedMovie.Title
		if m.selectedMovie.Year > 0 {
			title = fmt.Sprintf("%s (%d)", title, m.selectedMovie.Year)
		}
	}

	watched := "non"
	if m.selectedEntry.Watched {
		watched = "oui"
	}

	message := m.message
	if message == "" {
		message = "Modifie les champs puis valide avec Entrée."
	}

	return panelStyle.Render(strings.Join([]string{
		activeStyle.Render(title),
		subtleStyle.Render("Vu : " + watched),
		"",
		label("Note /10", m.detailFocus == 0),
		m.ratingInput.View(),
		"",
		label("Date de visionnage", m.detailFocus == 1),
		m.watchedAtInput.View(),
		"",
		label("Critique", m.detailFocus == 2),
		m.reviewInput.View(),
		"",
		statusLine(message),
	}, "\n"))
}

func (m Model) statsView() string {
	s := m.stats
	lines := []string{
		activeStyle.Render("Statistiques"),
		"",
		fmt.Sprintf("%-22s %d", "Films suivis", s.TotalMovies),
		fmt.Sprintf("%-22s %d", "Films vus", s.TotalWatched),
		fmt.Sprintf("%-22s %d", "Films notés", s.TotalRated),
	}

	if s.TotalRated > 0 {
		lines = append(lines, fmt.Sprintf("%-22s %.1f / 10", "Note moyenne", s.AverageRating))
	} else {
		lines = append(lines, fmt.Sprintf("%-22s %s", "Note moyenne", subtleStyle.Render("—")))
	}

	if len(s.BestMovies) > 0 {
		lines = append(lines, "", labelStyle.Render("Meilleur(s) film(s)"))
		for _, mr := range s.BestMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s %.1f / 10", title, mr.Rating))
		}
	}

	if len(s.WorstMovies) > 0 {
		lines = append(lines, "", labelStyle.Render("Film(s) les moins aimés"))
		for _, mr := range s.WorstMovies {
			title := mr.Movie.Title
			if mr.Movie.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, mr.Movie.Year)
			}
			lines = append(lines, fmt.Sprintf("  %-32s %.1f / 10", title, mr.Rating))
		}
	}

	if len(s.ByMonth) > 0 {
		lines = append(lines, "", labelStyle.Render("Visionnages par mois"))
		lines = append(lines, asciiHistogram(s.ByMonth))
	}

	if s.TotalMovies == 0 {
		lines = append(lines, "", subtleStyle.Render("Ajoute des films pour voir tes statistiques."))
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
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
		"Entrée      sélectionner, ajouter ou enregistrer",
		"↑/↓         naviguer dans les listes",
		"a           ajouter un film depuis la liste",
		"d           supprimer le film sélectionné",
		"Tab         changer de champ",
		"w           marquer vu aujourd'hui",
		"u           marquer non vu",
		"Esc ou m    revenir en arrière",
		"s           paramètres",
		"l           connexion",
		"? ou h      aide",
		"q           quitter",
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

func statusLine(message string) string {
	if message == "" {
		return ""
	}

	lower := strings.ToLower(message)
	if strings.Contains(lower, "invalide") ||
		strings.Contains(lower, "impossible") ||
		strings.Contains(lower, "obligatoire") ||
		strings.Contains(lower, "ne peut pas") ||
		strings.Contains(lower, "saisis") {
		return errorStyle.Render(message)
	}
	return activeStyle.Render(message)
}
