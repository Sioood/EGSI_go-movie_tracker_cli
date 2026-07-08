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
		Title: "Synchronisation",
		Shortcuts: []Shortcut{
			{Keys: "S", Action: "lancer une synchronisation manuelle"},
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
	var lines []string
	lines = append(lines, activeStyle.Render(messages.UI.HelpTitle), "")

	for _, section := range helpSections {
		lines = append(lines, labelStyle.Render(section.Title))
		for _, sc := range section.Shortcuts {
			lines = append(lines, fmt.Sprintf("  %-12s %s", sc.Keys, sc.Action))
		}
		lines = append(lines, "")
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}
