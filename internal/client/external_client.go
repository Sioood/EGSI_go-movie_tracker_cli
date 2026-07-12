package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/tmdb"
)

// ExternalClient performs TMDB search via the MovieTracker server proxy.
type ExternalClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewExternalClient creates an ExternalClient with a 10s timeout.
func NewExternalClient(baseURL string) *ExternalClient {
	return &ExternalClient{
		BaseURL: normalizeBaseURL(baseURL),
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetBaseURL updates the API base URL.
func (c *ExternalClient) SetBaseURL(baseURL string) {
	c.BaseURL = normalizeBaseURL(baseURL)
}

func (c *ExternalClient) client() *http.Client {
	return httpClient(c.HTTPClient)
}

// SearchMovies queries GET /api/v1/search/external.
func (c *ExternalClient) SearchMovies(ctx context.Context, accessToken, query string, year int) ([]tmdb.SearchResult, error) {
	values := url.Values{}
	values.Set("q", query)
	if year > 0 {
		values.Set("year", strconv.Itoa(year))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/search/external?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read external search response: %v", apperrors.ErrNetwork, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, apperrors.ErrUnauthorized
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("%w: recherche TMDB indisponible sur le serveur", apperrors.ErrNetwork)
	}
	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = fmt.Sprintf("external search failed (%d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: %s", apperrors.ErrNetwork, message)
	}

	var payload tmdb.SearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse external search response: %w", err)
	}
	return payload.Results, nil
}
