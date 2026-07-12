package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/transport/syncdto"
)

// SyncPayload is the bulk sync export/import body.
type SyncPayload = syncdto.Payload

// ImportResult is returned by POST /api/v1/sync.
type ImportResult = syncdto.ImportResult

// SyncClient performs HTTP calls against the MovieTracker sync API.
type SyncClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewSyncClient creates a SyncClient with a 10s timeout.
func NewSyncClient(baseURL string) *SyncClient {
	return &SyncClient{
		BaseURL:    normalizeBaseURL(baseURL),
		HTTPClient: &http.Client{Timeout: defaultTimeout},
	}
}

// SetBaseURL updates the API base URL (e.g. after a settings change).
func (c *SyncClient) SetBaseURL(baseURL string) {
	c.BaseURL = normalizeBaseURL(baseURL)
}

func (c *SyncClient) client() *http.Client {
	return httpClient(c.HTTPClient)
}

// Export downloads the authenticated user's dataset.
func (c *SyncClient) Export(ctx context.Context, accessToken string) (SyncPayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/sync", nil)
	if err != nil {
		return SyncPayload{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client().Do(req)
	if err != nil {
		return SyncPayload{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return SyncPayload{}, apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return SyncPayload{}, mapAPIError(resp)
	}

	var payload SyncPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SyncPayload{}, fmt.Errorf("%w: decode export: %v", apperrors.ErrNetwork, err)
	}
	return payload, nil
}

// Import uploads a dataset to the server.
func (c *SyncClient) Import(ctx context.Context, accessToken string, payload SyncPayload) (ImportResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return ImportResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/sync", bytes.NewReader(data))
	if err != nil {
		return ImportResult{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client().Do(req)
	if err != nil {
		return ImportResult{}, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ImportResult{}, apperrors.ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return ImportResult{}, mapAPIError(resp)
	}

	var result ImportResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ImportResult{}, fmt.Errorf("%w: decode import: %v", apperrors.ErrNetwork, err)
	}
	return result, nil
}
