package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.PasswordHash, formatTime(user.CreatedAt), formatTime(user.UpdatedAt))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.User{}, fmt.Errorf("%w: %s", apperrors.ErrEmailAlreadyExists, user.Email)
		}
		return domain.User{}, fmt.Errorf("%w: create user: %w", apperrors.ErrDB, err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = ?
	`, email)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	user, err := scanUser(row)
	if errors.Is(err, apperrors.ErrUserNotFound) {
		return domain.User{}, fmt.Errorf("%w: id %s", apperrors.ErrUserNotFound, id)
	}
	return user, err
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(s userScanner) (domain.User, error) {
	var user domain.User
	var createdAt, updatedAt string

	err := s.Scan(&user.ID, &user.Email, &user.PasswordHash, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, apperrors.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: scan user: %w", apperrors.ErrDB, err)
	}

	ct, err := parseTime(createdAt)
	if err != nil {
		return domain.User{}, err
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return domain.User{}, err
	}
	user.CreatedAt = ct
	user.UpdatedAt = ut
	return user, nil
}
