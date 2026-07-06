package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/auth"
	"github.com/movietracker/movie-tracker/internal/domain"
)

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// UserStore is the persistence interface required by AuthService.
type UserStore interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
}

// TokenPair holds the access and refresh JWT strings returned after a successful auth.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService struct {
	users  UserStore
	secret []byte
}

func NewAuthService(users UserStore, secret []byte) *AuthService {
	return &AuthService{users: users, secret: secret}
}

// Register creates a new account and returns a token pair.
func (s *AuthService) Register(ctx context.Context, email, password string) (TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateEmail(email); err != nil {
		return TokenPair{}, err
	}
	if err := validatePassword(password); err != nil {
		return TokenPair{}, err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, domain.User{Email: email, PasswordHash: hash})
	if err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(user)
}

// Login validates credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		// Constant-time-ish: don't reveal whether the email exists.
		return TokenPair{}, apperrors.ErrInvalidCredentials
	}

	match, err := auth.ComparePassword(password, user.PasswordHash)
	if err != nil || !match {
		return TokenPair{}, apperrors.ErrInvalidCredentials
	}

	return s.issueTokens(user)
}

// Refresh validates a refresh token and issues a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := auth.ParseToken(refreshToken, s.secret)
	if err != nil {
		return TokenPair{}, apperrors.ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return TokenPair{}, apperrors.ErrInvalidCredentials
	}

	return s.issueTokens(user)
}

func (s *AuthService) issueTokens(user domain.User) (TokenPair, error) {
	access, err := auth.GenerateAccessToken(user.ID, user.Email, s.secret)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := auth.GenerateRefreshToken(user.ID, user.Email, s.secret)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func validateEmail(email string) error {
	if !emailRE.MatchString(email) {
		return fmt.Errorf("%w: email invalide", apperrors.ErrValidation)
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: mot de passe trop court (min 8 caractères)", apperrors.ErrValidation)
	}
	return nil
}
