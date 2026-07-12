package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/tmdb"
)

func TestExternalClientSearchMovies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/external" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("q"); got != "inception" {
			t.Fatalf("query: want inception, got %q", got)
		}
		if got := r.URL.Query().Get("year"); got != "2010" {
			t.Fatalf("year: want 2010, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(tmdb.SearchResponse{
			Results: []tmdb.SearchResult{{ID: 1, Title: "Inception", Year: 2010}},
		})
	}))
	defer srv.Close()

	c := client.NewExternalClient(srv.URL)
	results, err := c.SearchMovies(context.Background(), "access-token", "inception", 2010)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Inception" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestExternalClientUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := client.NewExternalClient(srv.URL)
	_, err := c.SearchMovies(context.Background(), "bad-token", "test", 0)
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestExternalClientUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := client.NewExternalClient(srv.URL)
	_, err := c.SearchMovies(context.Background(), "", "test", 0)
	if err == nil || !errors.Is(err, apperrors.ErrNetwork) {
		t.Fatalf("want network error, got %v", err)
	}
}

func TestExternalClientNetworkError(t *testing.T) {
	c := client.NewExternalClient("http://127.0.0.1:1")
	_, err := c.SearchMovies(context.Background(), "", "test", 0)
	if !errors.Is(err, apperrors.ErrNetwork) {
		t.Fatalf("want ErrNetwork, got %v", err)
	}
}

func TestExternalClientSetBaseURL(t *testing.T) {
	c := client.NewExternalClient("http://localhost:8080/")
	c.SetBaseURL("http://example.com")
	if c.BaseURL != "http://example.com" {
		t.Fatalf("expected normalized base URL, got %q", c.BaseURL)
	}
}
