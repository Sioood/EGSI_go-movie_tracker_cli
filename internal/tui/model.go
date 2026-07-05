package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type MovieClient interface {
	CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	GetMovie(ctx context.Context, id string) (domain.Movie, error)
	ListMovies(ctx context.Context, userID string) ([]domain.Movie, error)
	SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error)
	UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error)
	DeleteMovie(ctx context.Context, id string) error
	SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error)
	GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error)
}

type Model struct {
	route          Route
	previous       Route
	state          AppState
	width          int
	height         int
	service        MovieClient
	menu           list.Model
	movies         list.Model
	movieRecords   []domain.Movie
	watchEntries   map[string]domain.WatchEntry
	selectedMovie  domain.Movie
	selectedEntry  domain.WatchEntry
	themeInput     textinput.Model
	emailInput     textinput.Model
	titleInput     textinput.Model
	yearInput      textinput.Model
	searchInput    textinput.Model
	ratingInput    textinput.Model
	watchedAtInput textinput.Model
	reviewInput    textarea.Model
	formFocus      int
	detailFocus    int
	filter         domain.MovieFilter
	sort           domain.MovieSort
	message        string
}

func New(services ...MovieClient) Model {
	var movieService MovieClient
	if len(services) > 0 {
		movieService = services[0]
	}

	menu := list.New(mainMenuItems(), list.NewDefaultDelegate(), 0, 0)
	menu.Title = "Menu principal"
	menu.SetShowStatusBar(false)
	menu.SetFilteringEnabled(false)

	movies := list.New(nil, list.NewDefaultDelegate(), 0, 0)
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

	titleInput := textinput.New()
	titleInput.Placeholder = "Titre du film"
	titleInput.CharLimit = 120

	yearInput := textinput.New()
	yearInput.Placeholder = "2026"
	yearInput.CharLimit = 4

	searchInput := textinput.New()
	searchInput.Placeholder = "Rechercher un titre..."
	searchInput.CharLimit = 80

	ratingInput := textinput.New()
	ratingInput.Placeholder = "8.5"
	ratingInput.CharLimit = 4

	watchedAtInput := textinput.New()
	watchedAtInput.Placeholder = "YYYY-MM-DD"
	watchedAtInput.CharLimit = 10

	reviewInput := textarea.New()
	reviewInput.Placeholder = "Votre critique..."
	reviewInput.SetWidth(64)
	reviewInput.SetHeight(6)

	model := Model{
		route:          RouteSplash,
		state:          defaultState(),
		service:        movieService,
		menu:           menu,
		movies:         movies,
		watchEntries:   make(map[string]domain.WatchEntry),
		themeInput:     themeInput,
		emailInput:     emailInput,
		titleInput:     titleInput,
		yearInput:      yearInput,
		searchInput:    searchInput,
		ratingInput:    ratingInput,
		watchedAtInput: watchedAtInput,
		reviewInput:    reviewInput,
		filter:         domain.MovieFilterAll,
		sort:           domain.MovieSortTitle,
	}
	model.refreshMovies()
	return model
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
	if m.route == RouteMovieList && m.searchInput.Focused() {
		if msg.String() == "esc" {
			m.searchInput.Blur()
			return m, nil
		}
		return m.updateMovieList(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.route == RouteMovieForm || m.route == RouteMovieDetail {
			m.goTo(RouteMovieList)
			return m, nil
		}
		if m.route != RouteMainMenu && m.route != RouteSplash {
			m.goTo(RouteMainMenu)
			return m, nil
		}
	}

	switch m.route {
	case RouteMovieForm:
		return m.updateMovieForm(msg)
	case RouteSettings:
		return m.updateSettings(msg)
	case RouteLogin:
		return m.updateLogin(msg)
	case RouteMovieDetail:
		if m.reviewInput.Focused() && msg.String() != "tab" && msg.String() != "shift+tab" && msg.String() != "enter" {
			var cmd tea.Cmd
			m.reviewInput, cmd = m.reviewInput.Update(msg)
			return m, cmd
		}
		return m.updateMovieDetail(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
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
	if m.searchInput.Focused() {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.refreshMovies()
		return m, cmd
	}

	switch msg.String() {
	case "/":
		m.searchInput.Focus()
		return m, textinput.Blink
	case "f":
		m.filter = nextMovieFilter(m.filter)
		m.message = "Filtre : " + filterLabel(m.filter)
		m.refreshMovies()
		return m, nil
	case "t":
		m.sort = nextMovieSort(m.sort)
		m.message = "Tri : " + sortLabel(m.sort)
		m.refreshMovies()
		return m, nil
	case "c":
		m.searchInput.SetValue("")
		m.filter = domain.MovieFilterAll
		m.sort = domain.MovieSortTitle
		m.message = "Recherche et filtres réinitialisés."
		m.refreshMovies()
		return m, nil
	case "a":
		m.prepareMovieForm()
		m.goTo(RouteMovieForm)
		return m, nil
	case "enter":
		if item, ok := m.movies.SelectedItem().(movieItem); ok {
			m.openMovieDetail(item.movie)
			m.goTo(RouteMovieDetail)
		}
		return m, nil
	case "d":
		if item, ok := m.movies.SelectedItem().(movieItem); ok {
			if err := m.service.DeleteMovie(context.Background(), item.movie.ID); err != nil {
				m.message = "Suppression impossible : " + err.Error()
				return m, nil
			}
			m.message = "Film supprimé."
			m.refreshMovies()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.movies, cmd = m.movies.Update(msg)
	return m, cmd
}

func (m Model) updateMovieForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab":
		m.formFocus = (m.formFocus + 1) % 2
		m.focusMovieForm()
		return m, nil
	case "enter":
		movie, err := m.createMovieFromForm()
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.message = "Film ajouté : " + movie.Title
		m.refreshMovies()
		m.openMovieDetail(movie)
		m.goTo(RouteMovieDetail)
		return m, nil
	}

	var cmd tea.Cmd
	if m.formFocus == 0 {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.yearInput, cmd = m.yearInput.Update(msg)
	}
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
	switch msg.String() {
	case "tab", "shift+tab":
		m.detailFocus = (m.detailFocus + 1) % 3
		m.focusMovieDetail()
		return m, nil
	case "w":
		today := time.Now().Format("2006-01-02")
		m.watchedAtInput.SetValue(today)
		m.selectedEntry.Watched = true
		m.message = "Film marqué comme vu aujourd'hui."
		return m, nil
	case "u":
		m.watchedAtInput.SetValue("")
		m.selectedEntry.Watched = false
		m.message = "Film marqué comme non vu."
		return m, nil
	case "enter":
		if err := m.saveMovieDetail(); err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.message = "Détail enregistré."
		m.refreshMovies()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.detailFocus {
	case 0:
		m.ratingInput, cmd = m.ratingInput.Update(msg)
	case 1:
		m.watchedAtInput, cmd = m.watchedAtInput.Update(msg)
	default:
		m.reviewInput, cmd = m.reviewInput.Update(msg)
	}
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
	case RouteMovieForm:
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	case RouteMovieDetail:
		var cmd tea.Cmd
		m.reviewInput, cmd = m.reviewInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) goTo(route Route) {
	m.previous = m.route
	m.route = route
	m.clearFocus()

	switch route {
	case RouteMovieList:
		m.refreshMovies()
	case RouteSettings:
		m.themeInput.Focus()
	case RouteLogin:
		m.emailInput.Focus()
	case RouteMovieForm:
		m.focusMovieForm()
	case RouteMovieDetail:
		m.focusMovieDetail()
	}
}

func (m *Model) clearFocus() {
	m.themeInput.Blur()
	m.emailInput.Blur()
	m.titleInput.Blur()
	m.yearInput.Blur()
	m.searchInput.Blur()
	m.ratingInput.Blur()
	m.watchedAtInput.Blur()
	m.reviewInput.Blur()
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
	m.reviewInput.SetWidth(listWidth)
}

func (m *Model) refreshMovies() {
	if m.service == nil {
		m.movies.SetItems(nil)
		return
	}

	movies, err := m.service.SearchMovies(context.Background(), domain.MovieSearchParams{
		UserID: m.state.User.ID,
		Query:  m.searchInput.Value(),
		Filter: m.filter,
		Sort:   m.sort,
	})
	if err != nil {
		m.message = "Chargement impossible : " + err.Error()
		return
	}

	m.movieRecords = movies
	m.watchEntries = make(map[string]domain.WatchEntry, len(movies))

	items := make([]list.Item, 0, len(movies))
	for _, movie := range movies {
		entry, err := m.service.GetWatchEntry(context.Background(), movie.ID)
		if err != nil && !errors.Is(err, apperrors.ErrWatchEntryNotFound) {
			m.message = "Statut incomplet : " + err.Error()
		}
		if err == nil {
			m.watchEntries[movie.ID] = entry
		}
		items = append(items, movieItem{movie: movie, status: movieStatus(entry, err == nil)})
	}

	m.movies.SetItems(items)
}

func (m *Model) prepareMovieForm() {
	m.titleInput.SetValue("")
	m.yearInput.SetValue("")
	m.formFocus = 0
	m.message = ""
}

func (m *Model) focusMovieForm() {
	m.clearFocus()
	if m.formFocus == 0 {
		m.titleInput.Focus()
	} else {
		m.yearInput.Focus()
	}
}

func (m *Model) createMovieFromForm() (domain.Movie, error) {
	if m.service == nil {
		return domain.Movie{}, fmt.Errorf("service films indisponible")
	}

	year, err := parseOptionalYear(m.yearInput.Value())
	if err != nil {
		return domain.Movie{}, err
	}

	return m.service.CreateMovie(context.Background(), domain.Movie{
		UserID: m.state.User.ID,
		Title:  m.titleInput.Value(),
		Year:   year,
	})
}

func (m *Model) openMovieDetail(movie domain.Movie) {
	m.selectedMovie = movie
	entry := m.watchEntries[movie.ID]
	entry.MovieID = movie.ID
	m.selectedEntry = entry

	if entry.Rating != nil {
		m.ratingInput.SetValue(strconv.FormatFloat(*entry.Rating, 'f', -1, 64))
	} else {
		m.ratingInput.SetValue("")
	}

	if entry.WatchedAt != nil {
		m.watchedAtInput.SetValue(entry.WatchedAt.Format("2006-01-02"))
	} else {
		m.watchedAtInput.SetValue("")
	}

	m.reviewInput.SetValue(entry.Review)
	m.detailFocus = 0
	m.message = ""
}

func (m *Model) focusMovieDetail() {
	m.clearFocus()
	switch m.detailFocus {
	case 0:
		m.ratingInput.Focus()
	case 1:
		m.watchedAtInput.Focus()
	default:
		m.reviewInput.Focus()
	}
}

func (m *Model) saveMovieDetail() error {
	if m.service == nil {
		return fmt.Errorf("service films indisponible")
	}
	if m.selectedMovie.ID == "" {
		return fmt.Errorf("aucun film sélectionné")
	}

	rating, err := parseOptionalRating(m.ratingInput.Value())
	if err != nil {
		return err
	}

	watchedAt, err := parseOptionalDate(m.watchedAtInput.Value())
	if err != nil {
		return err
	}

	entry := m.selectedEntry
	entry.MovieID = m.selectedMovie.ID
	entry.Rating = rating
	entry.RatingScale = 10
	entry.Review = strings.TrimSpace(m.reviewInput.Value())
	entry.WatchedAt = watchedAt
	entry.Watched = watchedAt != nil || entry.Watched

	saved, err := m.service.SaveWatchEntry(context.Background(), entry)
	if err != nil {
		return err
	}
	m.selectedEntry = saved
	m.watchEntries[m.selectedMovie.ID] = saved
	return nil
}

func mainMenuItems() []list.Item {
	return []list.Item{
		menuItem{"Films", "Parcourir et gérer la liste locale", RouteMovieList},
		menuItem{"Statistiques", "Voir les indicateurs de suivi", RouteStats},
		menuItem{"Paramètres", "Changer le thème et les préférences", RouteSettings},
		menuItem{"Connexion", "Préparer l'authentification serveur", RouteLogin},
		menuItem{"Aide", "Afficher les raccourcis", RouteHelp},
	}
}

func movieStatus(entry domain.WatchEntry, found bool) string {
	if !found || !entry.Watched {
		return "non vu"
	}

	parts := []string{"vu"}
	if entry.WatchedAt != nil {
		parts = append(parts, "le "+entry.WatchedAt.Format("2006-01-02"))
	}
	if entry.Rating != nil {
		parts = append(parts, fmt.Sprintf("note %.1f/10", *entry.Rating))
	}
	return strings.Join(parts, " · ")
}

func nextMovieFilter(filter domain.MovieFilter) domain.MovieFilter {
	switch filter {
	case domain.MovieFilterAll:
		return domain.MovieFilterWatched
	case domain.MovieFilterWatched:
		return domain.MovieFilterUnwatched
	case domain.MovieFilterUnwatched:
		return domain.MovieFilterRated
	case domain.MovieFilterRated:
		return domain.MovieFilterUnrated
	default:
		return domain.MovieFilterAll
	}
}

func filterLabel(filter domain.MovieFilter) string {
	switch filter {
	case domain.MovieFilterWatched:
		return "vus"
	case domain.MovieFilterUnwatched:
		return "non vus"
	case domain.MovieFilterRated:
		return "notés"
	case domain.MovieFilterUnrated:
		return "sans note"
	default:
		return "tous"
	}
}

func nextMovieSort(sort domain.MovieSort) domain.MovieSort {
	switch sort {
	case domain.MovieSortTitle:
		return domain.MovieSortDate
	case domain.MovieSortDate:
		return domain.MovieSortRating
	default:
		return domain.MovieSortTitle
	}
}

func sortLabel(sort domain.MovieSort) string {
	switch sort {
	case domain.MovieSortDate:
		return "date"
	case domain.MovieSortRating:
		return "note"
	default:
		return "titre"
	}
}

func parseOptionalYear(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	year, err := strconv.Atoi(value)
	if err != nil || year < 0 {
		return 0, fmt.Errorf("année invalide")
	}
	return year, nil
}

func parseOptionalRating(value string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	rating, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("note invalide")
	}
	return &rating, nil
}

func parseOptionalDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("date invalide, format attendu YYYY-MM-DD")
	}
	return &parsed, nil
}
