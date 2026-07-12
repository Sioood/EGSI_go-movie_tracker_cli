package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

const (
	defaultBaseURL = "https://api.themoviedb.org/3"
	defaultTimeout = 10 * time.Second
)

// Client calls the TMDB HTTP API.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Language   string
}

// NewClient creates a TMDB client.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:   strings.TrimSpace(apiKey),
		BaseURL:  defaultBaseURL,
		Language: "fr-FR",
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultTimeout}
}

// SearchMovies queries TMDB /search/movie.
func (c *Client) SearchMovies(ctx context.Context, query string, year int) ([]SearchResult, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("%w: TMDB API key is not configured", apperrors.ErrValidation)
	}

	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, fmt.Errorf("%w: search query is too short", apperrors.ErrValidation)
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("language", c.Language)
	values.Set("include_adult", "false")
	if year > 0 {
		values.Set("year", strconv.Itoa(year))
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + "/search/movie?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrNetwork, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read tmdb response: %w", apperrors.ErrNetwork, err)
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: tmdb unavailable (%d)", apperrors.ErrNetwork, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: invalid TMDB API key", apperrors.ErrUnauthorized)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: tmdb search failed (%d)", apperrors.ErrNetwork, resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			Overview    string `json:"overview"`
			PosterPath  string `json:"poster_path"`
			ReleaseDate string `json:"release_date"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse tmdb response: %w", err)
	}

	results := make([]SearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		results = append(results, SearchResult{
			ID:         item.ID,
			Title:      item.Title,
			Year:       parseReleaseYear(item.ReleaseDate),
			Overview:   item.Overview,
			PosterPath: item.PosterPath,
		})
	}
	return results, nil
}

func parseReleaseYear(releaseDate string) int {
	if len(releaseDate) < 4 {
		return 0
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil {
		return 0
	}
	return year
}
