package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
)

// newFullRouter creates a router backed by an in-memory DB with all migrations applied.
func newFullRouter(t *testing.T) http.Handler {
	t.Helper()

	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, "full_"+t.Name())

	dsn := "file:" + name + "?mode=memory&cache=shared&_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ServerMigrations, "migrations/server")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)
	watchRepo := repository.NewWatchEntryRepository(db)
	statsRepo := repository.NewStatsRepository(db)

	return server.NewRouter(server.Services{
		Auth:   service.NewAuthService(userRepo, testSecret),
		Movies: service.NewMovieService(movieRepo, watchRepo),
		Stats:  service.NewStatsService(statsRepo),
	}, testSecret)
}

// registerAndLogin is a test helper that registers a unique user and returns their access token.
func registerAndLogin(t *testing.T, router http.Handler, suffix string) string {
	t.Helper()
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email":    fmt.Sprintf("user%s@test.com", suffix),
		"password": "password123",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d: %s", rr.Code, rr.Body)
	}
	var pair service.TokenPair
	json.NewDecoder(rr.Body).Decode(&pair)
	return pair.AccessToken
}

func authGet(t *testing.T, router http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func authPost(t *testing.T, router http.Handler, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func authPut(t *testing.T, router http.Handler, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func authDelete(t *testing.T, router http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// --- Movie CRUD ---

func TestMovieCreateAndGet(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "create")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{
		"title": "Arrival",
		"year":  2016,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d: %s", rr.Code, rr.Body)
	}

	var created struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr.Body).Decode(&created)
	if created.Movie.ID == "" || created.Movie.Title != "Arrival" {
		t.Fatalf("unexpected created movie: %+v", created.Movie)
	}

	rr2 := authGet(t, router, "/api/v1/movies/"+created.Movie.ID, token)
	if rr2.Code != http.StatusOK {
		t.Fatalf("get: want 200, got %d", rr2.Code)
	}
}

func TestMovieList(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "list")

	authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Heat", "year": 1995})
	authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Inception", "year": 2010})

	rr := authGet(t, router, "/api/v1/movies", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: want 200, got %d: %s", rr.Code, rr.Body)
	}

	var resp struct {
		Movies []any `json:"movies"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Movies) != 2 {
		t.Fatalf("want 2 movies, got %d", len(resp.Movies))
	}
}

func TestMovieUpdate(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "update")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Old Title", "year": 2000})
	var created struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr.Body).Decode(&created)

	rr2 := authPut(t, router, "/api/v1/movies/"+created.Movie.ID, token, map[string]any{
		"title": "New Title", "year": 2001,
	})
	if rr2.Code != http.StatusOK {
		t.Fatalf("update: want 200, got %d: %s", rr2.Code, rr2.Body)
	}

	var updated struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr2.Body).Decode(&updated)
	if updated.Movie.Title != "New Title" {
		t.Fatalf("want title 'New Title', got %q", updated.Movie.Title)
	}
}

func TestMovieDelete(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "delete")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Doomed", "year": 2020})
	var created struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr.Body).Decode(&created)

	rr2 := authDelete(t, router, "/api/v1/movies/"+created.Movie.ID, token)
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", rr2.Code)
	}

	rr3 := authGet(t, router, "/api/v1/movies/"+created.Movie.ID, token)
	if rr3.Code != http.StatusNotFound {
		t.Fatalf("after delete: want 404, got %d", rr3.Code)
	}
}

func TestMovieForbiddenCrossUser(t *testing.T) {
	router := newFullRouter(t)
	tokenA := registerAndLogin(t, router, "crossA")
	tokenB := registerAndLogin(t, router, "crossB")

	rr := authPost(t, router, "/api/v1/movies", tokenA, map[string]any{"title": "Private", "year": 2020})
	var created struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr.Body).Decode(&created)

	rr2 := authGet(t, router, "/api/v1/movies/"+created.Movie.ID, tokenB)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("cross-user get: want 403, got %d", rr2.Code)
	}

	rr3 := authDelete(t, router, "/api/v1/movies/"+created.Movie.ID, tokenB)
	if rr3.Code != http.StatusForbidden {
		t.Fatalf("cross-user delete: want 403, got %d", rr3.Code)
	}
}

// --- Watch entry ---

func TestWatchEntry(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "watch")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Arrival", "year": 2016})
	var created struct {
		Movie domain.Movie `json:"movie"`
	}
	json.NewDecoder(rr.Body).Decode(&created)

	rating := 9.0
	rr2 := authPut(t, router, "/api/v1/movies/"+created.Movie.ID+"/watch", token, map[string]any{
		"watched":    true,
		"rating":     rating,
		"watched_at": "2026-07-01",
		"review":     "Excellent.",
	})
	if rr2.Code != http.StatusOK {
		t.Fatalf("watch: want 200, got %d: %s", rr2.Code, rr2.Body)
	}

	var resp struct {
		WatchEntry *domain.WatchEntry `json:"watch_entry"`
	}
	json.NewDecoder(rr2.Body).Decode(&resp)
	if resp.WatchEntry == nil || !resp.WatchEntry.Watched || resp.WatchEntry.Review != "Excellent." {
		t.Fatalf("unexpected watch entry: %+v", resp.WatchEntry)
	}
	if resp.WatchEntry.Rating == nil || *resp.WatchEntry.Rating != rating {
		t.Fatalf("unexpected rating: %+v", resp.WatchEntry.Rating)
	}
}

func TestWatchEntryInvalidDate(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "watchdate")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Test", "year": 2020})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr.Body).Decode(&created)

	rr2 := authPut(t, router, "/api/v1/movies/"+created.Movie.ID+"/watch", token, map[string]any{
		"watched": true, "watched_at": "not-a-date",
	})
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("invalid date: want 400, got %d", rr2.Code)
	}
}

// --- Stats ---

func TestStatsEndpoint(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "stats")

	// Empty stats
	rr := authGet(t, router, "/api/v1/stats", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("stats: want 200, got %d: %s", rr.Code, rr.Body)
	}

	var s domain.Stats
	json.NewDecoder(rr.Body).Decode(&s)
	if s.TotalMovies != 0 {
		t.Fatalf("want 0 total movies, got %d", s.TotalMovies)
	}

	// Add a movie and watch it
	rr2 := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Heat", "year": 1995})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr2.Body).Decode(&created)

	rating := 8.0
	authPut(t, router, "/api/v1/movies/"+created.Movie.ID+"/watch", token, map[string]any{
		"watched": true, "rating": rating, "watched_at": "2026-06-15",
	})

	rr3 := authGet(t, router, "/api/v1/stats", token)
	json.NewDecoder(rr3.Body).Decode(&s)
	if s.TotalMovies != 1 || s.TotalWatched != 1 || s.TotalRated != 1 {
		t.Fatalf("unexpected stats after watch: %+v", s)
	}
	if s.AverageRating != 8.0 {
		t.Fatalf("want avg 8.0, got %f", s.AverageRating)
	}
}

func TestStatsRequiresAuth(t *testing.T) {
	rr := authGet(t, newFullRouter(t), "/api/v1/stats", "bad-token")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

// --- Sync ---

func TestSyncExportEmpty(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "syncexp")

	rr := authGet(t, router, "/api/v1/sync", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("sync export: want 200, got %d: %s", rr.Code, rr.Body)
	}

	var payload struct {
		Movies       []domain.Movie      `json:"movies"`
		WatchEntries []domain.WatchEntry `json:"watch_entries"`
	}
	json.NewDecoder(rr.Body).Decode(&payload)
	if len(payload.Movies) != 0 {
		t.Fatalf("want 0 movies, got %d", len(payload.Movies))
	}
}

func TestSyncRoundtrip(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "syncrt")

	// Create two movies + one watch entry on the server.
	rr1 := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Film A", "year": 2020})
	rr2 := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Film B", "year": 2021})
	var mA, mB struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr1.Body).Decode(&mA)
	json.NewDecoder(rr2.Body).Decode(&mB)

	authPut(t, router, "/api/v1/movies/"+mA.Movie.ID+"/watch", token, map[string]any{
		"watched": true, "rating": 7.5, "watched_at": "2026-05-01",
	})

	// Export via GET /api/v1/sync.
	rrExport := authGet(t, router, "/api/v1/sync", token)
	if rrExport.Code != http.StatusOK {
		t.Fatalf("export: want 200, got %d", rrExport.Code)
	}

	var exported struct {
		Movies       []domain.Movie      `json:"movies"`
		WatchEntries []domain.WatchEntry `json:"watch_entries"`
	}
	json.NewDecoder(rrExport.Body).Decode(&exported)

	if len(exported.Movies) != 2 {
		t.Fatalf("export: want 2 movies, got %d", len(exported.Movies))
	}
	if len(exported.WatchEntries) != 1 {
		t.Fatalf("export: want 1 watch entry, got %d", len(exported.WatchEntries))
	}

	// Import the exported data via POST /api/v1/sync (idempotent re-import).
	rrImport := authPost(t, router, "/api/v1/sync", token, map[string]any{
		"movies":        exported.Movies,
		"watch_entries": exported.WatchEntries,
	})
	if rrImport.Code != http.StatusOK {
		t.Fatalf("import: want 200, got %d: %s", rrImport.Code, rrImport.Body)
	}

	var importResult map[string]int
	json.NewDecoder(rrImport.Body).Decode(&importResult)
	if importResult["synced_movies"] != 2 {
		t.Fatalf("import: want synced_movies=2, got %d", importResult["synced_movies"])
	}
}

func TestSyncImportRejectsOtherUsersMovies(t *testing.T) {
	router := newFullRouter(t)
	tokenA := registerAndLogin(t, router, "syncOwnerA")
	tokenB := registerAndLogin(t, router, "syncOwnerB")

	// User A creates a movie.
	rr := authPost(t, router, "/api/v1/movies", tokenA, map[string]any{"title": "A Movie", "year": 2020})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr.Body).Decode(&created)

	// User B tries to sync a watch entry for user A's movie.
	// The sync should not crash and the watch entry should be silently ignored.
	rrImport := authPost(t, router, "/api/v1/sync", tokenB, map[string]any{
		"movies": []any{},
		"watch_entries": []map[string]any{
			{"movie_id": created.Movie.ID, "watched": true},
		},
	})
	if rrImport.Code != http.StatusOK {
		t.Fatalf("import: want 200, got %d", rrImport.Code)
	}

	var result map[string]int
	json.NewDecoder(rrImport.Body).Decode(&result)
	if result["synced_watch_entries"] != 0 {
		t.Fatalf("cross-user watch entry should be rejected, got synced=%d", result["synced_watch_entries"])
	}
}

func TestSyncDeleteMovies(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "syncdel")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "To Delete", "year": 2020})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr.Body).Decode(&created)

	rrImport := authPost(t, router, "/api/v1/sync", token, map[string]any{
		"movies":            []any{},
		"watch_entries":     []any{},
		"deleted_movie_ids": []string{created.Movie.ID},
	})
	if rrImport.Code != http.StatusOK {
		t.Fatalf("import delete: want 200, got %d: %s", rrImport.Code, rrImport.Body)
	}

	rrGet := authGet(t, router, "/api/v1/movies/"+created.Movie.ID, token)
	if rrGet.Code != http.StatusNotFound {
		t.Fatalf("expected deleted movie 404, got %d", rrGet.Code)
	}
}

func TestSyncImportLWWSkipsStaleMovie(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "synclww")

	rr := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Current", "year": 2020})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr.Body).Decode(&created)

	stale := created.Movie
	stale.Title = "Stale"
	stale.UpdatedAt = created.Movie.UpdatedAt.Add(-2 * time.Hour)

	rrImport := authPost(t, router, "/api/v1/sync", token, map[string]any{
		"movies":        []domain.Movie{stale},
		"watch_entries": []any{},
	})
	if rrImport.Code != http.StatusOK {
		t.Fatalf("import stale: want 200, got %d", rrImport.Code)
	}

	rrGet := authGet(t, router, "/api/v1/movies/"+created.Movie.ID, token)
	var payload struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rrGet.Body).Decode(&payload)
	if payload.Movie.Title != "Current" {
		t.Fatalf("stale import should not overwrite title, got %q", payload.Movie.Title)
	}
}

func TestMoviesRequireAuth(t *testing.T) {
	router := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without token, got %d", rr.Code)
	}
}

func TestMovieListFilterAndSort(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "filtersort")

	authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Zulu", "year": 2020})
	rrB := authPost(t, router, "/api/v1/movies", token, map[string]any{"title": "Alpha", "year": 2021})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rrB.Body).Decode(&created)

	authPut(t, router, "/api/v1/movies/"+created.Movie.ID+"/watch", token, map[string]any{
		"watched": true, "rating": 8.0, "watched_at": "2026-01-15",
	})

	rrWatched := authGet(t, router, "/api/v1/movies?filter=watched", token)
	if rrWatched.Code != http.StatusOK {
		t.Fatalf("watched filter: want 200, got %d", rrWatched.Code)
	}
	var watchedPayload struct {
		Movies []struct {
			Movie domain.Movie `json:"movie"`
		} `json:"movies"`
	}
	json.NewDecoder(rrWatched.Body).Decode(&watchedPayload)
	if len(watchedPayload.Movies) != 1 {
		t.Fatalf("watched filter: want 1 movie, got %d", len(watchedPayload.Movies))
	}
	if watchedPayload.Movies[0].Movie.Title != "Alpha" {
		t.Fatalf("watched filter: want Alpha, got %q", watchedPayload.Movies[0].Movie.Title)
	}

	rrSearch := authGet(t, router, "/api/v1/movies?q=Zulu&sort=title", token)
	if rrSearch.Code != http.StatusOK {
		t.Fatalf("search: want 200, got %d", rrSearch.Code)
	}
	var searchPayload struct {
		Movies []struct {
			Movie domain.Movie `json:"movie"`
		} `json:"movies"`
	}
	json.NewDecoder(rrSearch.Body).Decode(&searchPayload)
	if len(searchPayload.Movies) != 1 || searchPayload.Movies[0].Movie.Title != "Zulu" {
		t.Fatalf("search q=Zulu: unexpected result %+v", searchPayload.Movies)
	}
}

func TestMovieForbiddenCrossUserUpdate(t *testing.T) {
	router := newFullRouter(t)
	tokenA := registerAndLogin(t, router, "updA")
	tokenB := registerAndLogin(t, router, "updB")

	rr := authPost(t, router, "/api/v1/movies", tokenA, map[string]any{"title": "Owned", "year": 2020})
	var created struct{ Movie domain.Movie `json:"movie"` }
	json.NewDecoder(rr.Body).Decode(&created)

	rrPut := authPut(t, router, "/api/v1/movies/"+created.Movie.ID, tokenB, map[string]any{
		"title": "Hijacked", "year": 2020,
	})
	if rrPut.Code != http.StatusForbidden {
		t.Fatalf("cross-user update: want 403, got %d", rrPut.Code)
	}
}
