package tui

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

func TestKeyboardNavigationBetweenScreens(t *testing.T) {
	store := newFakeMovieService()
	store.movies = []domain.Movie{{ID: "movie-1", UserID: "local-user", Title: "Arrival", Year: 2016}}
	model := testNewModel(store)

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMainMenu)

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMovieList)

	model = press(t, model, "enter")
	assertRoute(t, model, RouteMovieDetail)

	model = press(t, model, "esc")
	assertRoute(t, model, RouteMovieList)

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

func TestMovieAddAndDetailSave(t *testing.T) {
	store := newFakeMovieService()
	model := testNewModel(store)
	model.goTo(RouteMovieList)

	model = press(t, model, "a")
	assertRoute(t, model, RouteMovieForm)

	model.titleInput.SetValue("Heat")
	model.yearInput.SetValue("1995")
	model = press(t, model, "enter")
	assertRoute(t, model, RouteMovieDetail)

	if len(store.movies) != 1 {
		t.Fatalf("expected one created movie, got %d", len(store.movies))
	}
	if store.movies[0].Title != "Heat" || store.movies[0].Year != 1995 {
		t.Fatalf("unexpected created movie: %+v", store.movies[0])
	}

	model.ratingInput.SetValue("9")
	model.watchedAtInput.SetValue("2026-07-05")
	model.reviewInput.SetValue("Excellent polar.")
	model = press(t, model, "enter")

	entry := store.entries[store.movies[0].ID]
	if !entry.Watched || entry.Rating == nil || *entry.Rating != 9 || entry.Review != "Excellent polar." {
		t.Fatalf("unexpected saved watch entry: %+v", entry)
	}
	if entry.WatchedAt == nil || entry.WatchedAt.Format("2006-01-02") != "2026-07-05" {
		t.Fatalf("unexpected watched date: %+v", entry.WatchedAt)
	}
}

func TestMovieSearchAndFilters(t *testing.T) {
	store := newFakeMovieService()
	store.movies = []domain.Movie{
		{ID: "movie-1", UserID: "local-user", Title: "Arrival", Year: 2016},
		{ID: "movie-2", UserID: "local-user", Title: "Heat", Year: 1995},
	}
	rating := 9.0
	store.entries["movie-2"] = domain.WatchEntry{MovieID: "movie-2", Watched: true, Rating: &rating}

	model := testNewModel(store)
	model.goTo(RouteMovieList)

	model = press(t, model, "/")
	model = press(t, model, "a")
	model = press(t, model, "r")

	if len(model.movieRecords) != 1 || model.movieRecords[0].Title != "Arrival" {
		t.Fatalf("expected search to keep Arrival only, got %+v", model.movieRecords)
	}

	model.searchInput.SetValue("")
	model.refreshMovies()
	model = press(t, model, "esc")
	model = press(t, model, "f")

	if model.filter != domain.MovieFilterWatched {
		t.Fatalf("expected watched filter, got %s", model.filter)
	}
	if len(model.movieRecords) != 1 || model.movieRecords[0].Title != "Heat" {
		t.Fatalf("expected watched filter to keep Heat only, got %+v", model.movieRecords)
	}

	model = press(t, model, "t")
	if model.sort != domain.MovieSortDate {
		t.Fatalf("expected date sort, got %s", model.sort)
	}
}

func TestTypingShortcutLettersInMovieFormDoesNotNavigate(t *testing.T) {
	model := testNewModel(newFakeMovieService())
	model.goTo(RouteMovieList)
	model = press(t, model, "a")
	assertRoute(t, model, RouteMovieForm)

	model = press(t, model, "m")
	model = press(t, model, "q")
	model = press(t, model, "s")
	model = press(t, model, "l")

	assertRoute(t, model, RouteMovieForm)
	if model.titleInput.Value() != "mqsl" {
		t.Fatalf("expected shortcut letters to be typed, got %q", model.titleInput.Value())
	}
}

func TestLoginSuccess(t *testing.T) {
	auth := &fakeAuthClient{
		pair: service.TokenPair{AccessToken: "access", RefreshToken: "refresh"},
		user: UserInfo{ID: "user-1", Email: "alice@example.com"},
	}
	var saved SessionState
	model := testNewModelWithAuth(newFakeMovieService(), auth, func(state SessionState) error {
		saved = state
		return nil
	})
	model.goTo(RouteLogin)

	model.emailInput.SetValue("alice@example.com")
	model.passwordInput.SetValue("secret123")
	model.loginFocus = 1
	model = press(t, model, "enter")

	updated, cmd := model.Update(authResultMsg{
		session: sessionFromTokens(auth.pair, auth.user),
		action:  "login",
	})
	model, _ = updated.(Model)
	_ = cmd

	if !model.state.Session.Authenticated || model.state.Session.Email != "alice@example.com" {
		t.Fatalf("expected authenticated session, got %+v", model.state.Session)
	}
	if model.state.Config.OfflineMode {
		t.Fatal("expected offline mode disabled after login")
	}
	if saved.Email != "alice@example.com" {
		t.Fatalf("expected session saved, got %+v", saved)
	}
}

func TestRegisterNavigation(t *testing.T) {
	model := testNewModel(newFakeMovieService())
	model.goTo(RouteLogin)
	model = press(t, model, "r")
	assertRoute(t, model, RouteRegister)
	model = press(t, model, "esc")
	assertRoute(t, model, RouteLogin)
}

func TestLoginScreenWhenAuthenticatedOnlyAllowsLogout(t *testing.T) {
	auth := &fakeAuthClient{}
	var cleared bool
	model := testNewModelWithAuth(newFakeMovieService(), auth, nil)
	model.clearSession = func() error {
		cleared = true
		return nil
	}
	model.state.Session = SessionState{
		AccessToken:   "access",
		RefreshToken:  "refresh",
		ServerUserID:  "user-1",
		Email:         "alice@example.com",
		Authenticated: true,
	}
	model.goTo(RouteLogin)

	model = press(t, model, "r")
	assertRoute(t, model, RouteLogin)
	if model.state.Session.Authenticated == false {
		t.Fatal("expected r not to logout")
	}

	model = press(t, model, "enter")
	assertRoute(t, model, RouteLogin)
	if auth.loginCalls != 0 {
		t.Fatalf("expected no login call while authenticated, got %d", auth.loginCalls)
	}

	model = press(t, model, "d")
	assertRoute(t, model, RouteLogin)
	if model.state.Session.Authenticated {
		t.Fatal("expected session cleared after d")
	}
	if !cleared {
		t.Fatal("expected persisted session cleared")
	}
}

func TestSettingsThemeCycle(t *testing.T) {
	model := testNewModel(newFakeMovieService())
	model.goTo(RouteSettings)

	if model.state.Config.Theme != "midnight" {
		t.Fatalf("expected default midnight theme, got %s", model.state.Config.Theme)
	}

	model = press(t, model, "right")
	if model.state.Config.Theme != "solar" {
		t.Fatalf("expected solar theme after right, got %s", model.state.Config.Theme)
	}
}

func TestSettingsBackupExportRequiresAuth(t *testing.T) {
	model := testNewModel(newFakeMovieService())
	model.goTo(RouteSettings)
	model.state.Config.OfflineMode = true

	model = press(t, model, "e")
	if model.messageKind != messages.KindError {
		t.Fatalf("expected error when offline, got %s", model.message)
	}
}

func testNewModel(store MovieClient) Model {
	return New(Options{MovieService: store})
}

func testNewModelWithAuth(store MovieClient, auth AuthClient, saveSession func(SessionState) error) Model {
	return New(Options{
		MovieService: store,
		Auth:         auth,
		SaveSession:  saveSession,
		SaveConfig:   func(Config) error { return nil },
	})
}

type fakeAuthClient struct {
	pair       service.TokenPair
	user       UserInfo
	err        error
	loginCalls int
}

func (f *fakeAuthClient) Register(ctx context.Context, email, password string) (service.TokenPair, error) {
	if f.err != nil {
		return service.TokenPair{}, f.err
	}
	return f.pair, nil
}

func (f *fakeAuthClient) Login(ctx context.Context, email, password string) (service.TokenPair, error) {
	f.loginCalls++
	if f.err != nil {
		return service.TokenPair{}, f.err
	}
	return f.pair, nil
}

func (f *fakeAuthClient) Refresh(ctx context.Context, refreshToken string) (service.TokenPair, error) {
	return f.pair, f.err
}

func (f *fakeAuthClient) Me(ctx context.Context, accessToken string) (UserInfo, error) {
	if f.err != nil {
		return UserInfo{}, f.err
	}
	return f.user, nil
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

type fakeMovieService struct {
	movies  []domain.Movie
	entries map[string]domain.WatchEntry
}

func newFakeMovieService() *fakeMovieService {
	return &fakeMovieService{entries: make(map[string]domain.WatchEntry)}
}

func (s *fakeMovieService) CreateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	if movie.ID == "" {
		movie.ID = "movie-" + strconv.Itoa(len(s.movies)+1)
	}
	s.movies = append(s.movies, movie)
	return movie, nil
}

func (s *fakeMovieService) GetMovie(ctx context.Context, id string) (domain.Movie, error) {
	for _, movie := range s.movies {
		if movie.ID == id {
			return movie, nil
		}
	}
	return domain.Movie{}, apperrors.ErrMovieNotFound
}

func (s *fakeMovieService) ListMovies(ctx context.Context, userID string) ([]domain.Movie, error) {
	return s.SearchMovies(ctx, domain.MovieSearchParams{UserID: userID})
}

func (s *fakeMovieService) SearchMovies(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	var result []domain.Movie
	for _, movie := range s.movies {
		if movie.UserID != params.UserID {
			continue
		}
		if params.Query != "" && !strings.Contains(strings.ToLower(movie.Title), strings.ToLower(params.Query)) {
			continue
		}
		entry, hasEntry := s.entries[movie.ID]
		if !matchesFakeFilter(entry, hasEntry, params.Filter) {
			continue
		}
		result = append(result, movie)
	}
	return result, nil
}

func (s *fakeMovieService) UpdateMovie(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	for index := range s.movies {
		if s.movies[index].ID == movie.ID {
			s.movies[index] = movie
			return movie, nil
		}
	}
	return domain.Movie{}, apperrors.ErrMovieNotFound
}

func (s *fakeMovieService) DeleteMovie(ctx context.Context, id string) error {
	for index := range s.movies {
		if s.movies[index].ID == id {
			s.movies = append(s.movies[:index], s.movies[index+1:]...)
			delete(s.entries, id)
			return nil
		}
	}
	return apperrors.ErrMovieNotFound
}

func (s *fakeMovieService) SaveWatchEntry(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	if entry.ID == "" {
		entry.ID = "entry-" + entry.MovieID
	}
	if entry.WatchedAt != nil {
		parsed, _ := time.Parse("2006-01-02", entry.WatchedAt.Format("2006-01-02"))
		entry.WatchedAt = &parsed
	}
	s.entries[entry.MovieID] = entry
	return entry, nil
}

func (s *fakeMovieService) GetWatchEntry(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	entry, ok := s.entries[movieID]
	if !ok {
		return domain.WatchEntry{}, apperrors.ErrWatchEntryNotFound
	}
	return entry, nil
}

func (s *fakeMovieService) GetStats(ctx context.Context, userID string) (domain.Stats, error) {
	stats := domain.Stats{TotalMovies: len(s.movies)}
	for _, e := range s.entries {
		if e.Watched {
			stats.TotalWatched++
		}
		if e.Rating != nil {
			stats.TotalRated++
			stats.AverageRating += *e.Rating
		}
	}
	if stats.TotalRated > 0 {
		stats.AverageRating /= float64(stats.TotalRated)
	}
	return stats, nil
}

func matchesFakeFilter(entry domain.WatchEntry, found bool, filter domain.MovieFilter) bool {
	switch filter {
	case domain.MovieFilterWatched:
		return found && entry.Watched
	case domain.MovieFilterUnwatched:
		return !found || !entry.Watched
	case domain.MovieFilterRated:
		return found && entry.Rating != nil
	case domain.MovieFilterUnrated:
		return !found || entry.Rating == nil
	default:
		return true
	}
}
