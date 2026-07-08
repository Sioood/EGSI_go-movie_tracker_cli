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
	"github.com/movietracker/movie-tracker/internal/service"
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
	GetStats(ctx context.Context, userID string) (domain.Stats, error)
}

// AuthClient performs remote authentication (implemented by internal/client).
type AuthClient interface {
	Register(ctx context.Context, email, password string) (service.TokenPair, error)
	Login(ctx context.Context, email, password string) (service.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (service.TokenPair, error)
	Me(ctx context.Context, accessToken string) (UserInfo, error)
}

// UserInfo is the authenticated user profile from the server.
type UserInfo struct {
	ID    string
	Email string
}

// Options configures the TUI model at startup.
type Options struct {
	MovieService  MovieClient
	Auth          AuthClient
	SyncRunner    SyncRunner
	ResolveUserID func() string
	State         AppState
	SaveConfig    func(Config) error
	SaveSession   func(SessionState) error
	ClearSession  func() error
}

type authResultMsg struct {
	session SessionState
	err     error
	action  string
}

type Model struct {
	route                Route
	previous             Route
	state                AppState
	width                int
	height               int
	service              MovieClient
	auth                 AuthClient
	saveConfig           func(Config) error
	saveSession          func(SessionState) error
	clearSession         func() error
	syncRunner           SyncRunner
	resolveUserID        func() string
	syncStatus           SyncStatus
	pendingCount         int
	lastSyncAt           time.Time
	syncSyncing          bool
	syncError            string
	menu                 list.Model
	movies               list.Model
	movieRecords         []domain.Movie
	watchEntries         map[string]domain.WatchEntry
	selectedMovie        domain.Movie
	selectedEntry        domain.WatchEntry
	themeInput           textinput.Model
	serverURLInput       textinput.Model
	emailInput           textinput.Model
	passwordInput        textinput.Model
	confirmPasswordInput textinput.Model
	titleInput           textinput.Model
	yearInput            textinput.Model
	searchInput          textinput.Model
	ratingInput          textinput.Model
	watchedAtInput       textinput.Model
	reviewInput          textarea.Model
	formFocus            int
	detailFocus          int
	loginFocus           int
	registerFocus        int
	settingsFocus        int
	filter               domain.MovieFilter
	sort                 domain.MovieSort
	stats                domain.Stats
	message              string
	authLoading          bool
}

func New(opts Options) Model {
	movieService := opts.MovieService

	menu := list.New(mainMenuItems(), list.NewDefaultDelegate(), 0, 0)
	menu.Title = "Menu principal"
	menu.SetShowStatusBar(false)
	menu.SetFilteringEnabled(false)

	movies := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	movies.Title = "Films"
	movies.SetShowStatusBar(false)
	movies.SetFilteringEnabled(false)

	state := opts.State
	if state.Config.Theme == "" {
		state = defaultState()
	}

	themeInput := newTextInput("midnight", 32)
	themeInput.SetValue(state.Config.Theme)

	serverURLInput := newTextInput("http://localhost:8080", 120)
	serverURLInput.SetValue(state.Config.ServerURL)

	emailInput := newTextInput("vous@example.com", 80)
	passwordInput := newPasswordInput("mot de passe", 64)
	confirmPasswordInput := newPasswordInput("confirmer", 64)

	titleInput := newTextInput("Titre du film", 120)
	yearInput := newTextInput("2026", 4)
	searchInput := newTextInput("Rechercher un titre...", 80)
	ratingInput := newTextInput("8.5", 4)
	watchedAtInput := newTextInput("YYYY-MM-DD", 10)

	reviewInput := textarea.New()
	reviewInput.Placeholder = "Votre critique..."
	reviewInput.SetWidth(64)
	reviewInput.SetHeight(6)

	model := Model{
		route:                RouteSplash,
		state:                state,
		service:              movieService,
		auth:                 opts.Auth,
		saveConfig:           opts.SaveConfig,
		saveSession:          opts.SaveSession,
		clearSession:         opts.ClearSession,
		syncRunner:           opts.SyncRunner,
		resolveUserID:        opts.ResolveUserID,
		syncStatus:           SyncStatusIdle,
		menu:                 menu,
		movies:               movies,
		watchEntries:         make(map[string]domain.WatchEntry),
		themeInput:           themeInput,
		serverURLInput:       serverURLInput,
		emailInput:           emailInput,
		passwordInput:        passwordInput,
		confirmPasswordInput: confirmPasswordInput,
		titleInput:           titleInput,
		yearInput:            yearInput,
		searchInput:          searchInput,
		ratingInput:          ratingInput,
		watchedAtInput:       watchedAtInput,
		reviewInput:          reviewInput,
		filter:               domain.MovieFilterAll,
		sort:                 domain.MovieSortTitle,
	}
	model.refreshMovies()
	model.refreshPendingCount()
	return model
}

func newTextInput(placeholder string, limit int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = limit
	return input
}

func newPasswordInput(placeholder string, limit int) textinput.Model {
	input := newTextInput(placeholder, limit)
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	return input
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	if tick := m.scheduleSyncTick(); tick != nil {
		cmds = append(cmds, tick)
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
		return m, nil
	case authResultMsg:
		return m.handleAuthResult(msg)
	case SyncRequestMsg:
		return m.startSync()
	case syncTickMsg:
		return m.startSync()
	case syncResultMsg:
		return m.handleSyncResult(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateActiveBubble(msg)
}

func (m Model) handleAuthResult(msg authResultMsg) (tea.Model, tea.Cmd) {
	m.authLoading = false
	if msg.err != nil {
		m.message = authErrorMessage(msg.err)
		return m, nil
	}

	m.state.Session = msg.session
	m.state.Config.OfflineMode = false
	if m.saveSession != nil {
		_ = m.saveSession(msg.session)
	}
	if m.saveConfig != nil {
		_ = m.saveConfig(m.state.Config)
	}

	switch msg.action {
	case "login":
		m.message = "Connecté en tant que " + msg.session.Email
	case "register":
		m.message = "Compte créé pour " + msg.session.Email
	default:
		m.message = "Session restaurée pour " + msg.session.Email
	}
	m.goTo(RouteMainMenu)
	return m.startSync()
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
		if m.route == RouteRegister {
			m.goTo(RouteLogin)
			return m, nil
		}
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
	case RouteRegister:
		return m.updateRegister(msg)
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
		if m.route != RouteSplash && m.route != RouteLogin && m.route != RouteRegister && m.route != RouteSettings {
			m.goTo(RouteHelp)
			return m, nil
		}
	case "m":
		m.goTo(RouteMainMenu)
		return m, nil
	case "s":
		if m.route != RouteSplash && m.route != RouteSettings && m.route != RouteLogin && m.route != RouteRegister {
			m.goTo(RouteSettings)
			return m, nil
		}
	case "l":
		if m.route != RouteSplash && m.route != RouteLogin && m.route != RouteRegister && m.route != RouteSettings {
			m.goTo(RouteLogin)
			return m, nil
		}
	case "S":
		if m.route != RouteMovieForm && m.route != RouteLogin && m.route != RouteRegister && m.route != RouteSettings {
			if !m.searchInput.Focused() {
				return m.startSync()
			}
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
	case "tab", "shift+tab":
		m.settingsFocus = (m.settingsFocus + 1) % 2
		m.focusSettings()
		return m, nil
	case "o":
		m.state.Config.OfflineMode = !m.state.Config.OfflineMode
		m.message = fmt.Sprintf("Mode hors ligne : %t", m.state.Config.OfflineMode)
		if m.saveConfig != nil {
			_ = m.saveConfig(m.state.Config)
		}
		return m, nil
	case "d":
		if m.state.Session.Authenticated {
			m.state.Session = SessionState{}
			if m.clearSession != nil {
				_ = m.clearSession()
			}
			m.message = "Déconnecté."
		}
		return m, nil
	case "enter", "ctrl+s":
		return m.saveSettings()
	}

	var cmd tea.Cmd
	switch m.settingsFocus {
	case 0:
		m.themeInput, cmd = m.themeInput.Update(msg)
	default:
		m.serverURLInput, cmd = m.serverURLInput.Update(msg)
	}
	return m, cmd
}

func (m Model) saveSettings() (tea.Model, tea.Cmd) {
	theme := strings.TrimSpace(m.themeInput.Value())
	if theme == "" {
		m.message = "Le thème ne peut pas être vide."
		return m, nil
	}
	serverURL := strings.TrimSpace(m.serverURLInput.Value())
	if serverURL == "" {
		m.message = "L'URL du serveur ne peut pas être vide."
		return m, nil
	}

	m.state.Config.Theme = theme
	m.state.Config.ServerURL = serverURL
	if m.saveConfig != nil {
		if err := m.saveConfig(m.state.Config); err != nil {
			m.message = "Sauvegarde impossible : " + err.Error()
			return m, nil
		}
	}
	m.message = "Paramètres enregistrés."
	return m, nil
}

func (m Model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.authLoading {
		return m, nil
	}

	switch msg.String() {
	case "tab", "shift+tab":
		m.loginFocus = (m.loginFocus + 1) % 2
		m.focusLogin()
		return m, nil
	case "r":
		m.goTo(RouteRegister)
		return m, nil
	case "enter":
		if m.loginFocus < 1 {
			m.loginFocus++
			m.focusLogin()
			return m, nil
		}
		email := strings.TrimSpace(m.emailInput.Value())
		password := m.passwordInput.Value()
		if err := validateAuthInput(email, password); err != nil {
			m.message = err.Error()
			return m, nil
		}
		if m.auth == nil {
			m.message = "Client d'authentification indisponible."
			return m, nil
		}
		m.authLoading = true
		m.message = "Connexion en cours..."
		return m, loginCmd(m.auth, email, password)
	}

	var cmd tea.Cmd
	if m.loginFocus == 0 {
		m.emailInput, cmd = m.emailInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m Model) updateRegister(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.authLoading {
		return m, nil
	}

	switch msg.String() {
	case "tab", "shift+tab":
		m.registerFocus = (m.registerFocus + 1) % 3
		m.focusRegister()
		return m, nil
	case "enter":
		if m.registerFocus < 2 {
			m.registerFocus++
			m.focusRegister()
			return m, nil
		}
		email := strings.TrimSpace(m.emailInput.Value())
		password := m.passwordInput.Value()
		confirm := m.confirmPasswordInput.Value()
		if err := validateAuthInput(email, password); err != nil {
			m.message = err.Error()
			return m, nil
		}
		if password != confirm {
			m.message = "Les mots de passe ne correspondent pas."
			return m, nil
		}
		if m.auth == nil {
			m.message = "Client d'authentification indisponible."
			return m, nil
		}
		m.authLoading = true
		m.message = "Inscription en cours..."
		return m, registerCmd(m.auth, email, password)
	}

	var cmd tea.Cmd
	switch m.registerFocus {
	case 0:
		m.emailInput, cmd = m.emailInput.Update(msg)
	case 1:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	default:
		m.confirmPasswordInput, cmd = m.confirmPasswordInput.Update(msg)
	}
	return m, cmd
}

func loginCmd(auth AuthClient, email, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pair, err := auth.Login(ctx, email, password)
		if err != nil {
			return authResultMsg{err: err, action: "login"}
		}
		info, err := auth.Me(ctx, pair.AccessToken)
		if err != nil {
			return authResultMsg{err: err, action: "login"}
		}
		return authResultMsg{
			session: sessionFromTokens(pair, info),
			action:  "login",
		}
	}
}

func registerCmd(auth AuthClient, email, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pair, err := auth.Register(ctx, email, password)
		if err != nil {
			return authResultMsg{err: err, action: "register"}
		}
		info, err := auth.Me(ctx, pair.AccessToken)
		if err != nil {
			return authResultMsg{err: err, action: "register"}
		}
		return authResultMsg{
			session: sessionFromTokens(pair, info),
			action:  "register",
		}
	}
}

func sessionFromTokens(pair service.TokenPair, info UserInfo) SessionState {
	return SessionState{
		AccessToken:   pair.AccessToken,
		RefreshToken:  pair.RefreshToken,
		ServerUserID:  info.ID,
		Email:         info.Email,
		Authenticated: true,
	}
}

func validateAuthInput(email, password string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("l'email est requis")
	}
	if len(password) < 8 {
		return fmt.Errorf("le mot de passe doit contenir au moins 8 caractères")
	}
	return nil
}

func authErrorMessage(err error) string {
	switch {
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		return "Identifiants invalides."
	case errors.Is(err, apperrors.ErrEmailAlreadyExists):
		return "Email déjà utilisé."
	case errors.Is(err, apperrors.ErrValidation):
		return "Données invalides : " + err.Error()
	case errors.Is(err, apperrors.ErrNetwork):
		return "Erreur réseau : " + err.Error()
	default:
		return "Authentification impossible : " + err.Error()
	}
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
		if m.settingsFocus == 0 {
			m.themeInput, cmd = m.themeInput.Update(msg)
		} else {
			m.serverURLInput, cmd = m.serverURLInput.Update(msg)
		}
		return m, cmd
	case RouteLogin:
		var cmd tea.Cmd
		if m.loginFocus == 0 {
			m.emailInput, cmd = m.emailInput.Update(msg)
		} else {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		}
		return m, cmd
	case RouteRegister:
		var cmd tea.Cmd
		switch m.registerFocus {
		case 0:
			m.emailInput, cmd = m.emailInput.Update(msg)
		case 1:
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		default:
			m.confirmPasswordInput, cmd = m.confirmPasswordInput.Update(msg)
		}
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
	m.authLoading = false

	switch route {
	case RouteMovieList:
		m.refreshMovies()
	case RouteStats:
		m.refreshStats()
	case RouteSettings:
		m.themeInput.SetValue(m.state.Config.Theme)
		m.serverURLInput.SetValue(m.state.Config.ServerURL)
		m.settingsFocus = 0
		m.focusSettings()
	case RouteLogin:
		m.loginFocus = 0
		m.emailInput.SetValue("")
		m.passwordInput.SetValue("")
		m.focusLogin()
	case RouteRegister:
		m.registerFocus = 0
		m.emailInput.SetValue("")
		m.passwordInput.SetValue("")
		m.confirmPasswordInput.SetValue("")
		m.focusRegister()
	case RouteMovieForm:
		m.focusMovieForm()
	case RouteMovieDetail:
		m.focusMovieDetail()
	}
}

func (m *Model) clearFocus() {
	m.themeInput.Blur()
	m.serverURLInput.Blur()
	m.emailInput.Blur()
	m.passwordInput.Blur()
	m.confirmPasswordInput.Blur()
	m.titleInput.Blur()
	m.yearInput.Blur()
	m.searchInput.Blur()
	m.ratingInput.Blur()
	m.watchedAtInput.Blur()
	m.reviewInput.Blur()
}

func (m *Model) focusSettings() {
	m.clearFocus()
	if m.settingsFocus == 0 {
		m.themeInput.Focus()
	} else {
		m.serverURLInput.Focus()
	}
}

func (m *Model) focusLogin() {
	m.clearFocus()
	if m.loginFocus == 0 {
		m.emailInput.Focus()
	} else {
		m.passwordInput.Focus()
	}
}

func (m *Model) focusRegister() {
	m.clearFocus()
	switch m.registerFocus {
	case 0:
		m.emailInput.Focus()
	case 1:
		m.passwordInput.Focus()
	default:
		m.confirmPasswordInput.Focus()
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
	m.reviewInput.SetWidth(listWidth)
}

func (m *Model) refreshMovies() {
	if m.service == nil {
		m.movies.SetItems(nil)
		return
	}

	movies, err := m.service.SearchMovies(context.Background(), domain.MovieSearchParams{
		UserID: m.currentUserID(),
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

func (m *Model) refreshStats() {
	if m.service == nil {
		return
	}
	stats, err := m.service.GetStats(context.Background(), m.currentUserID())
	if err != nil {
		m.message = "Stats indisponibles : " + err.Error()
		return
	}
	m.stats = stats
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
		UserID: m.currentUserID(),
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
		menuItem{"Connexion", "Se connecter au serveur", RouteLogin},
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
