package tmdb

import "time"

// SearchResult is a normalized movie search hit from TMDB.
type SearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Overview    string `json:"overview"`
	PosterPath  string `json:"poster_path"`
}

// SearchResponse is returned by search endpoints.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// CachedPayload stores serialized metadata in the local cache.
type CachedPayload struct {
	Result    SearchResult `json:"result"`
	FetchedAt time.Time    `json:"fetched_at"`
}
