package service

import (
	"context"
	"errors"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/auth"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type memoryUserStore struct {
	byEmail map[string]domain.User
	byID    map[string]domain.User
}

func (m *memoryUserStore) Create(ctx context.Context, user domain.User) (domain.User, error) {
	if _, exists := m.byEmail[user.Email]; exists {
		return domain.User{}, apperrors.ErrEmailAlreadyExists
	}
	if user.ID == "" {
		user.ID = "user-" + user.Email
	}
	m.byEmail[user.Email] = user
	m.byID[user.ID] = user
	return user, nil
}

func (m *memoryUserStore) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return domain.User{}, apperrors.ErrUserNotFound
	}
	return user, nil
}

func (m *memoryUserStore) GetByID(ctx context.Context, id string) (domain.User, error) {
	user, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperrors.ErrUserNotFound
	}
	return user, nil
}

func TestAuthServiceRegisterLoginRefresh(t *testing.T) {
	secret := []byte("phase-5-auth-service-test-secret-key")
	store := &memoryUserStore{byEmail: make(map[string]domain.User), byID: make(map[string]domain.User)}
	svc := NewAuthService(store, secret)
	ctx := context.Background()

	pair, err := svc.Register(ctx, "Alice@Example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected token pair from register")
	}

	loginPair, err := svc.Login(ctx, "alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginPair.AccessToken == "" {
		t.Fatal("expected access token from login")
	}

	refreshPair, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshPair.AccessToken == "" || refreshPair.RefreshToken == "" {
		t.Fatal("expected refreshed tokens")
	}
}

func TestAuthServiceLoginInvalidCredentials(t *testing.T) {
	secret := []byte("phase-5-auth-service-test-secret-key")
	store := &memoryUserStore{byEmail: make(map[string]domain.User), byID: make(map[string]domain.User)}
	svc := NewAuthService(store, secret)
	ctx := context.Background()

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	store.byEmail["known@example.com"] = domain.User{ID: "user-1", Email: "known@example.com", PasswordHash: hash}
	store.byID["user-1"] = store.byEmail["known@example.com"]

	_, err = svc.Login(ctx, "unknown@example.com", "secret123")
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("unknown email: want ErrInvalidCredentials, got %v", err)
	}

	_, err = svc.Login(ctx, "known@example.com", "wrong-pass")
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("wrong password: want ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthServiceRegisterValidation(t *testing.T) {
	svc := NewAuthService(&memoryUserStore{byEmail: make(map[string]domain.User), byID: make(map[string]domain.User)}, []byte("phase-5-auth-service-test-secret-key"))
	ctx := context.Background()

	_, err := svc.Register(ctx, "not-an-email", "secret123")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("invalid email: want ErrValidation, got %v", err)
	}

	_, err = svc.Register(ctx, "valid@example.com", "short")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("short password: want ErrValidation, got %v", err)
	}
}

func TestAuthServiceRefreshInvalidToken(t *testing.T) {
	svc := NewAuthService(&memoryUserStore{byEmail: make(map[string]domain.User), byID: make(map[string]domain.User)}, []byte("phase-5-auth-service-test-secret-key"))
	_, err := svc.Refresh(context.Background(), "not-a-jwt")
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}
