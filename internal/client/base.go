package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

const defaultTimeout = 10 * time.Second

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/")
}

func httpClient(custom *http.Client) *http.Client {
	if custom != nil {
		return custom
	}
	return &http.Client{Timeout: defaultTimeout}
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
