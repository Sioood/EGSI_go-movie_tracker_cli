package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/auth"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func openServerTestDB(t *testing.T) *sql.DB {
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
		t.Fatalf("open server test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedServerUser(t *testing.T, db *sql.DB, userID, email string) {
	t.Helper()
	users := NewUserRepository(db)
	_, err := users.Create(context.Background(), domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func TestUserRepositoryCRUD(t *testing.T) {
	db := openServerTestDB(t)
	users := NewUserRepository(db)
	ctx := context.Background()

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	created, err := users.Create(ctx, domain.User{Email: "alice@example.com", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated user id")
	}

	byEmail, err := users.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if byEmail.ID != created.ID || byEmail.Email != "alice@example.com" {
		t.Fatalf("unexpected user by email: %+v", byEmail)
	}

	byID, err := users.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Email != "alice@example.com" {
		t.Fatalf("unexpected user by id: %+v", byID)
	}
}

func TestUserRepositoryDuplicateEmail(t *testing.T) {
	db := openServerTestDB(t)
	users := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := auth.HashPassword("secret123")
	_, err := users.Create(ctx, domain.User{Email: "bob@example.com", PasswordHash: hash})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = users.Create(ctx, domain.User{Email: "bob@example.com", PasswordHash: hash})
	if !errors.Is(err, apperrors.ErrEmailAlreadyExists) {
		t.Fatalf("want ErrEmailAlreadyExists, got %v", err)
	}
}

func TestUserRepositoryNotFound(t *testing.T) {
	db := openServerTestDB(t)
	users := NewUserRepository(db)
	ctx := context.Background()

	_, err := users.GetByEmail(ctx, "missing@example.com")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("get by email: want ErrUserNotFound, got %v", err)
	}

	_, err = users.GetByID(ctx, "missing-id")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("get by id: want ErrUserNotFound, got %v", err)
	}
}
