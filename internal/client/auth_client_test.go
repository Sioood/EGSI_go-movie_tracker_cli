package client_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
)

var testSecret = []byte("phase-8-client-test-secret")

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

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
	router := server.NewRouter(server.Services{Auth: authSvc}, testSecret)
	return httptest.NewServer(router)
}

func TestAuthClientRegisterAndLogin(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	ctx := context.Background()

	pair, err := c.Register(ctx, "alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected tokens from register")
	}

	info, err := c.Me(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if info.Email != "alice@example.com" || info.ID == "" {
		t.Fatalf("unexpected user info: %+v", info)
	}

	pair2, err := c.Login(ctx, "alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair2.AccessToken == "" {
		t.Fatal("expected access token from login")
	}
}

func TestAuthClientLoginInvalidCredentials(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	_, err := c.Login(context.Background(), "nobody@example.com", "wrongpass1")
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthClientRegisterDuplicate(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	ctx := context.Background()

	_, err := c.Register(ctx, "bob@example.com", "secret123")
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err = c.Register(ctx, "bob@example.com", "secret123")
	if !errors.Is(err, apperrors.ErrEmailAlreadyExists) {
		t.Fatalf("want ErrEmailAlreadyExists, got %v", err)
	}
}

func TestAuthClientRefresh(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	ctx := context.Background()

	pair, err := c.Register(ctx, "carol@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	pair2, err := c.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if pair2.AccessToken == "" || pair2.RefreshToken == "" {
		t.Fatal("expected refreshed tokens")
	}

	info, err := c.Me(ctx, pair2.AccessToken)
	if err != nil {
		t.Fatalf("me after refresh: %v", err)
	}
	if info.Email != "carol@example.com" {
		t.Fatalf("unexpected email: %s", info.Email)
	}
}

func TestAuthClientMeNoToken(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	_, err := c.Me(context.Background(), "invalid-token")
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestAuthClientNetworkError(t *testing.T) {
	c := client.NewAuthClient("http://127.0.0.1:1")
	_, err := c.Login(context.Background(), "a@b.com", "secret123")
	if !errors.Is(err, apperrors.ErrNetwork) {
		t.Fatalf("want ErrNetwork, got %v", err)
	}
}

func TestRestoreSession(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := client.NewAuthClient(srv.URL)
	ctx := context.Background()

	pair, err := c.Register(ctx, "dave@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	restored, err := client.RestoreSession(ctx, c, config.Session{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Session.Email != "dave@example.com" || restored.Session.ServerUserID == "" {
		t.Fatalf("unexpected restored session: %+v", restored.Session)
	}
}
