package tui

import (
	"fmt"
	"strings"

	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

// Shortcut documents a keyboard binding for the help screen.
type Shortcut struct {
	Keys    string
	Action  string
	Screens []Route
}

// HelpSection groups shortcuts under a title.
type HelpSection struct {
	Title     string
	Shortcuts []Shortcut
}

var helpSections = []HelpSection{
	{
		Title: "Navigation",
		Shortcuts: []Shortcut{
			{Keys: "Entrée", Action: "sélectionner, valider ou enregistrer"},
			{Keys: "↑/↓", Action: "naviguer dans les listes", Screens: []Route{RouteMainMenu, RouteMovieList}},
			{Keys: "Espace", Action: "sélectionner dans le menu", Screens: []Route{RouteMainMenu, RouteMovieList}},
			{Keys: "Esc", Action: "revenir en arrière"},
			{Keys: "m", Action: "retourner au menu principal"},
			{Keys: "? ou h", Action: "afficher l'aide"},
			{Keys: "q", Action: "quitter l'application"},
			{Keys: "ctrl+c", Action: "quitter immédiatement"},
		},
	},
	{
		Title: "Films",
		Shortcuts: []Shortcut{
			{Keys: "/", Action: "focus recherche", Screens: []Route{RouteMovieList}},
			{Keys: "a", Action: "ajouter un film", Screens: []Route{RouteMovieList}},
			{Keys: "d", Action: "supprimer le film sélectionné", Screens: []Route{RouteMovieList}},
			{Keys: "f", Action: "changer le filtre", Screens: []Route{RouteMovieList}},
			{Keys: "t", Action: "changer le tri", Screens: []Route{RouteMovieList}},
			{Keys: "c", Action: "réinitialiser recherche et filtres", Screens: []Route{RouteMovieList}},
		},
	},
	{
		Title: "Détail film",
		Shortcuts: []Shortcut{
			{Keys: "Tab", Action: "champ suivant", Screens: []Route{RouteMovieForm, RouteMovieDetail, RouteSettings, RouteLogin, RouteRegister}},
			{Keys: "ctrl+t", Action: "rechercher un film sur TMDB", Screens: []Route{RouteMovieForm}},
			{Keys: "w", Action: "marquer vu aujourd'hui", Screens: []Route{RouteMovieDetail}},
			{Keys: "u", Action: "marquer non vu", Screens: []Route{RouteMovieDetail}},
		},
	},
	{
		Title: "Compte",
		Shortcuts: []Shortcut{
			{Keys: "l", Action: "ouvrir la connexion"},
			{Keys: "r", Action: "passer à l'inscription", Screens: []Route{RouteLogin}},
			{Keys: "s", Action: "ouvrir les paramètres"},
			{Keys: "o", Action: "basculer le mode hors ligne", Screens: []Route{RouteSettings}},
			{Keys: "d", Action: "se déconnecter", Screens: []Route{RouteSettings}},
			{Keys: "ctrl+s", Action: "enregistrer les paramètres", Screens: []Route{RouteSettings}},
		},
	},
	{
		Title: "Paramètres et backup",
		Shortcuts: []Shortcut{
			{Keys: "←/→", Action: "changer le thème", Screens: []Route{RouteSettings}},
			{Keys: "e", Action: "exporter config + état vers le serveur", Screens: []Route{RouteSettings}},
			{Keys: "i", Action: "importer config + état depuis le serveur", Screens: []Route{RouteSettings}},
			{Keys: "E", Action: "exporter config + état en local (JSON)", Screens: []Route{RouteSettings}},
			{Keys: "j", Action: "exporter les films en JSON local", Screens: []Route{RouteSettings}},
			{Keys: "J", Action: "exporter les films en CSV local", Screens: []Route{RouteSettings}},
		},
	},
	{
		Title: "Synchronisation",
		Shortcuts: []Shortcut{
			{Keys: "S", Action: "lancer une synchronisation manuelle"},
			{Keys: "K", Action: "résoudre les conflits de synchronisation"},
		},
	},
}

// AllShortcuts returns every documented shortcut (flattened).
func AllShortcuts() []Shortcut {
	var all []Shortcut
	for _, section := range helpSections {
		all = append(all, section.Shortcuts...)
	}
	return all
}

func (m Model) helpView() string {
	s := m.styles
	var lines []string
	lines = append(lines, s.Active.Render(messages.UI.HelpTitle), "")

	for _, section := range helpSections {
		lines = append(lines, s.Label.Render(section.Title))
		for _, sc := range section.Shortcuts {
			lines = append(lines, fmt.Sprintf("  %-12s %s", sc.Keys, sc.Action))
		}
		lines = append(lines, "")
	}

	return s.Panel.Render(strings.Join(lines, "\n"))
}
