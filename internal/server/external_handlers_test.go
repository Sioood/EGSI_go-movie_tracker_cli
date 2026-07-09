package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/movietracker/movie-tracker/internal/tmdb"
)

type stubTMDBSearcher struct {
	results []tmdb.SearchResult
	err     error
	lastQ   string
	lastY   int
}

func (s *stubTMDBSearcher) SearchMovies(r *http.Request, query string, year int) ([]tmdb.SearchResult, error) {
	s.lastQ = query
	s.lastY = year
	return s.results, s.err
}

func TestExternalSearchHandler(t *testing.T) {
	handler := &externalHandler{
		tmdb: &stubTMDBSearcher{
			results: []tmdb.SearchResult{{
				ID:    27205,
				Title: "Inception",
				Year:  2010,
			}},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/external?q=inception&year=2010", nil)
	rec := httptest.NewRecorder()
	handler.search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload tmdb.SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Title != "Inception" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExternalSearchHandlerMissingQuery(t *testing.T) {
	handler := &externalHandler{tmdb: &stubTMDBSearcher{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/external", nil)
	rec := httptest.NewRecorder()
	handler.search(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestExternalSearchHandlerUnavailable(t *testing.T) {
	handler := &externalHandler{tmdb: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/external?q=test", nil)
	rec := httptest.NewRecorder()
	handler.search(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
