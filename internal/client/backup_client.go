package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/service"
)

// BackupClient performs HTTP calls against the backup API.
type BackupClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewBackupClient creates a BackupClient with a 10s timeout.
func NewBackupClient(baseURL string) *BackupClient {
	return &BackupClient{
		BaseURL: normalizeBaseURL(baseURL),
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetBaseURL updates the API base URL.
func (c *BackupClient) SetBaseURL(baseURL string) {
	c.BaseURL = normalizeBaseURL(baseURL)
}

// ExportSnapshot downloads the full config + state snapshot.
func (c *BackupClient) ExportSnapshot(ctx context.Context, accessToken string) (service.BackupSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/backup", nil)
	if err != nil {
		return service.BackupSnapshot{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return service.BackupSnapshot{}, fmt.Errorf("backup export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return service.BackupSnapshot{}, apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return service.BackupSnapshot{}, mapSyncAPIError(resp)
	}

	var snapshot service.BackupSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return service.BackupSnapshot{}, fmt.Errorf("decode backup: %w", err)
	}
	return snapshot, nil
}

// ImportSnapshot uploads the full config + state snapshot.
func (c *BackupClient) ImportSnapshot(ctx context.Context, accessToken string, snapshot service.BackupSnapshot) error {
	body, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal backup: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+"/api/v1/backup", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("backup import: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return mapSyncAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// ExportConfig downloads the remote user config.
func (c *BackupClient) ExportConfig(ctx context.Context, accessToken string) (config.Config, error) {
	return decodeBackupResource[config.Config](c, ctx, accessToken, "/api/v1/backup/config")
}

// ImportConfig uploads the user config.
func (c *BackupClient) ImportConfig(ctx context.Context, accessToken string, cfg config.Config) error {
	return putBackupResource(c, ctx, accessToken, "/api/v1/backup/config", cfg)
}

// ExportState downloads the remote UI state.
func (c *BackupClient) ExportState(ctx context.Context, accessToken string) (config.State, error) {
	return decodeBackupResource[config.State](c, ctx, accessToken, "/api/v1/backup/state")
}

// ImportState uploads the UI state.
func (c *BackupClient) ImportState(ctx context.Context, accessToken string, state config.State) error {
	return putBackupResource(c, ctx, accessToken, "/api/v1/backup/state", state)
}

func decodeBackupResource[T any](c *BackupClient, ctx context.Context, accessToken, path string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("backup get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return zero, apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return zero, mapSyncAPIError(resp)
	}

	var value T
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

func putBackupResource(c *BackupClient, ctx context.Context, accessToken, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("backup put %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return mapSyncAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
