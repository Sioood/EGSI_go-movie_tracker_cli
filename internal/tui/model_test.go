package tui

import (
	"context"
	"strconv"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestKeyboardNavigationBetweenScreens(t *testing.T) {
	store := newFakeMovieService()
	store.movies = []domain.Movie{{ID: "movie-1", UserID: "local-user", Title: "Arrival", Year: 2016}}
	model := New(store)

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
	model := New(store)
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
	var result []domain.Movie
	for _, movie := range s.movies {
		if movie.UserID == userID {
			result = append(result, movie)
		}
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
