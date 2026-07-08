package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/service"
)

const defaultTimeout = 10 * time.Second

// UserInfo is returned by GET /api/me.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// AuthClient performs HTTP calls against the MovieTracker auth API.
type AuthClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAuthClient creates an AuthClient with a 10s timeout.
func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *AuthClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultTimeout}
}

// Register creates a new account and returns a token pair.
func (c *AuthClient) Register(ctx context.Context, email, password string) (service.TokenPair, error) {
	return c.postTokenPair(ctx, "/api/register", map[string]string{
		"email":    email,
		"password": password,
	})
}

// Login validates credentials and returns a token pair.
func (c *AuthClient) Login(ctx context.Context, email, password string) (service.TokenPair, error) {
	return c.postTokenPair(ctx, "/api/login", map[string]string{
		"email":    email,
		"password": password,
	})
}

// Refresh exchanges a refresh token for a new token pair.
func (c *AuthClient) Refresh(ctx context.Context, refreshToken string) (service.TokenPair, error) {
	return c.postTokenPair(ctx, "/api/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
}

// Me returns the authenticated user profile.
func (c *AuthClient) Me(ctx context.Context, accessToken string) (UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/me", nil)
	if err != nil {
		return UserInfo{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client().Do(req)
	if err != nil {
		return UserInfo{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return UserInfo{}, apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return UserInfo{}, mapAPIError(resp)
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return UserInfo{}, fmt.Errorf("%w: decode response: %v", apperrors.ErrNetwork, err)
	}
	return info, nil
}

func (c *AuthClient) postTokenPair(ctx context.Context, path string, body any) (service.TokenPair, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return service.TokenPair{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return service.TokenPair{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client().Do(req)
	if err != nil {
		return service.TokenPair{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return service.TokenPair{}, apperrors.ErrInvalidCredentials
	}
	if resp.StatusCode == http.StatusConflict {
		return service.TokenPair{}, apperrors.ErrEmailAlreadyExists
	}
	if resp.StatusCode == http.StatusBadRequest {
		return service.TokenPair{}, apperrors.ErrValidation
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return service.TokenPair{}, mapAPIError(resp)
	}

	var pair service.TokenPair
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		return service.TokenPair{}, fmt.Errorf("%w: decode response: %v", apperrors.ErrNetwork, err)
	}
	return pair, nil
}

func mapAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %s", apperrors.ErrUnauthorized, payload.Error)
		case http.StatusBadRequest:
			return fmt.Errorf("%w: %s", apperrors.ErrValidation, payload.Error)
		default:
			return fmt.Errorf("%w: %s", apperrors.ErrNetwork, payload.Error)
		}
	}
	return fmt.Errorf("%w: HTTP %d", apperrors.ErrNetwork, resp.StatusCode)
}

// IsUnauthorized reports whether err is an unauthorized response from the API.
func IsUnauthorized(err error) bool {
	return errors.Is(err, apperrors.ErrUnauthorized) || errors.Is(err, apperrors.ErrInvalidCredentials)
}
