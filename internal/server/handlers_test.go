package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
)

var testSecret = []byte("phase-6-test-secret-key")

// newTestRouter creates an isolated in-memory DB and router for each test.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	// Replace characters that SQLite DSN parsing does not allow in the DB name.
	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, t.Name())

	dsn := "file:" + name + "?mode=memory&cache=shared&_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ServerMigrations, "migrations/server")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testSecret)
	return server.NewRouter(server.Services{Auth: authSvc}, testSecret)
}

func postJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func decodeTokenPair(t *testing.T, rr *httptest.ResponseRecorder) service.TokenPair {
	t.Helper()
	var pair service.TokenPair
	if err := json.NewDecoder(rr.Body).Decode(&pair); err != nil {
		t.Fatalf("decode token pair: %v", err)
	}
	return pair
}

// --- Register ---

func TestRegisterSuccess(t *testing.T) {
	rr := postJSON(t, newTestRouter(t), "/api/register", map[string]string{
		"email": "alice@example.com", "password": "secret123",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body)
	}
	pair := decodeTokenPair(t, rr)
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	router := newTestRouter(t)
	postJSON(t, router, "/api/register", map[string]string{
		"email": "bob@example.com", "password": "secret123",
	})
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "bob@example.com", "password": "other-pass",
	})
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rr.Code)
	}
}

func TestRegisterEmailCaseInsensitive(t *testing.T) {
	router := newTestRouter(t)
	postJSON(t, router, "/api/register", map[string]string{
		"email": "Carol@Example.COM", "password": "secret123",
	})
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "carol@example.com", "password": "other-pass",
	})
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409 for case-insensitive duplicate, got %d", rr.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	router := newTestRouter(t)
	cases := []struct{ email, password string }{
		{"not-an-email", "secret123"},
		{"valid@example.com", "short"},
		{"@nodomain", "secret123"},
	}
	for _, c := range cases {
		rr := postJSON(t, router, "/api/register", map[string]string{
			"email": c.email, "password": c.password,
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("register %q / %q: want 400, got %d", c.email, c.password, rr.Code)
		}
	}
}

func TestRegisterInvalidJSON(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for bad JSON, got %d", rr.Code)
	}
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	router := newTestRouter(t)
	postJSON(t, router, "/api/register", map[string]string{
		"email": "carol@example.com", "password": "mypassword",
	})
	rr := postJSON(t, router, "/api/login", map[string]string{
		"email": "carol@example.com", "password": "mypassword",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body)
	}
	pair := decodeTokenPair(t, rr)
	if pair.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	router := newTestRouter(t)
	postJSON(t, router, "/api/register", map[string]string{
		"email": "dave@example.com", "password": "mypassword",
	})
	rr := postJSON(t, router, "/api/login", map[string]string{
		"email": "dave@example.com", "password": "wrongpassword",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	rr := postJSON(t, newTestRouter(t), "/api/login", map[string]string{
		"email": "ghost@example.com", "password": "secret123",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

// --- Refresh ---

func TestRefreshSuccess(t *testing.T) {
	router := newTestRouter(t)
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "eve@example.com", "password": "mypassword",
	})
	pair := decodeTokenPair(t, rr)

	rr2 := postJSON(t, router, "/api/refresh", map[string]string{
		"refresh_token": pair.RefreshToken,
	})
	if rr2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr2.Code, rr2.Body)
	}
	pair2 := decodeTokenPair(t, rr2)
	if pair2.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	rr := postJSON(t, newTestRouter(t), "/api/refresh", map[string]string{
		"refresh_token": "not.a.jwt",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

// --- Protected /api/me ---

func TestMeSuccess(t *testing.T) {
	router := newTestRouter(t)
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "frank@example.com", "password": "mypassword",
	})
	pair := decodeTokenPair(t, rr)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr2.Code, rr2.Body)
	}
	var payload map[string]string
	json.NewDecoder(rr2.Body).Decode(&payload)
	if payload["email"] != "frank@example.com" {
		t.Fatalf("unexpected email in /me response: %v", payload)
	}
}

func TestMeNoToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rr := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestMeWrongSecret(t *testing.T) {
	router := newTestRouter(t)
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "grace@example.com", "password": "mypassword",
	})
	pair := decodeTokenPair(t, rr)

	// Sign a token with a different secret — it should be rejected.
	otherRouter := server.NewRouter(
		server.Services{Auth: service.NewAuthService(repository.NewUserRepository(nil), []byte("wrong-secret"))},
		[]byte("wrong-secret"),
	)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rr2 := httptest.NewRecorder()
	otherRouter.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for wrong-secret token, got %d", rr2.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	newTestRouter(t).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"version":"1.0.0"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAccessTokenCannotRefresh(t *testing.T) {
	router := newTestRouter(t)
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "refresh-guard@example.com", "password": "mypassword",
	})
	pair := decodeTokenPair(t, rr)

	rr2 := postJSON(t, router, "/api/refresh", map[string]string{
		"refresh_token": pair.AccessToken,
	})
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("access token on /api/refresh: want 401, got %d: %s", rr2.Code, rr2.Body)
	}
}

func TestRefreshTokenCannotAccessProtectedRoute(t *testing.T) {
	router := newTestRouter(t)
	rr := postJSON(t, router, "/api/register", map[string]string{
		"email": "access-guard@example.com", "password": "mypassword",
	})
	pair := decodeTokenPair(t, rr)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token on /api/me: want 401, got %d: %s", rr2.Code, rr2.Body)
	}
}

func TestRateLimitReturns429(t *testing.T) {
	router := newTestRouter(t)
	var lastCode int
	for i := 0; i < 25; i++ {
		rr := postJSON(t, router, "/api/login", map[string]string{
			"email": "rate@example.com", "password": "wrongpassword",
		})
		lastCode = rr.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("want 429 after burst, got %d", lastCode)
	}
}
