package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSearchMovies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/movie" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "Inception" {
			t.Fatalf("unexpected query: %s", got)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"id": 27205,
					"title": "Inception",
					"overview": "A thief who steals secrets.",
					"poster_path": "/poster.jpg",
					"release_date": "2010-07-16"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.BaseURL = server.URL

	results, err := client.SearchMovies(context.Background(), "Inception", 0)
	if err != nil {
		t.Fatalf("search movies: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != 27205 || results[0].Title != "Inception" || results[0].Year != 2010 {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestClientSearchMoviesMissingAPIKey(t *testing.T) {
	client := NewClient("")
	if _, err := client.SearchMovies(context.Background(), "Inception", 0); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestCachedPayloadRoundTrip(t *testing.T) {
	payload := CachedPayload{Result: SearchResult{ID: 1, Title: "Test"}}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded CachedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Result.Title != "Test" {
		t.Fatalf("unexpected title: %s", decoded.Result.Title)
	}
}
