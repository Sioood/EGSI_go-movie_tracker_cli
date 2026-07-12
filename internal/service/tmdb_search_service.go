package service

import (
	"context"
	"log/slog"

	"github.com/movietracker/movie-tracker/internal/logging"
	"github.com/movietracker/movie-tracker/internal/tmdb"
)

var tmdbLog = logging.New("tmdb")

// TMDBProxySearcher searches TMDB through the MovieTracker server proxy.
type TMDBProxySearcher interface {
	SearchMovies(ctx context.Context, accessToken, query string, year int) ([]tmdb.SearchResult, error)
}

// TMDBCacheStore persists TMDB metadata locally.
type TMDBCacheStore interface {
	Get(ctx context.Context, tmdbID int) (tmdb.SearchResult, bool, error)
	CacheSearchResults(ctx context.Context, results []tmdb.SearchResult) error
}

// TMDBSearchService searches TMDB via proxy or direct client and caches results.
type TMDBSearchService struct {
	proxy       TMDBProxySearcher
	direct      *tmdb.Client
	cache       TMDBCacheStore
	accessToken func() string
	useProxy    func() bool
}

// NewTMDBSearchService creates a TMDB search orchestrator for the CLI.
func NewTMDBSearchService(
	proxy TMDBProxySearcher,
	direct *tmdb.Client,
	cache TMDBCacheStore,
	accessToken func() string,
	useProxy func() bool,
) *TMDBSearchService {
	return &TMDBSearchService{
		proxy:       proxy,
		direct:      direct,
		cache:       cache,
		accessToken: accessToken,
		useProxy:    useProxy,
	}
}

// SearchMovies returns TMDB results using the best available backend.
func (s *TMDBSearchService) SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error) {
	var (
		results []tmdb.SearchResult
		err     error
	)

	if s.useProxy != nil && s.useProxy() && s.proxy != nil {
		token := ""
		if s.accessToken != nil {
			token = s.accessToken()
		}
		results, err = s.proxy.SearchMovies(ctx, token, query, year)
	} else if s.direct != nil && s.direct.APIKey != "" {
		results, err = s.direct.SearchMovies(ctx, query, year)
	} else if s.proxy != nil {
		token := ""
		if s.accessToken != nil {
			token = s.accessToken()
		}
		results, err = s.proxy.SearchMovies(ctx, token, query, year)
	} else {
		return nil, tmdb.ErrUnavailable
	}

	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err := s.cache.CacheSearchResults(ctx, results); err != nil {
			tmdbLog.Warn("cache TMDB search results", slog.Any("err", err))
		}
	}
	return results, nil
}
