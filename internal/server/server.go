package server

import (
	"net/http"

	"golang.org/x/time/rate"

	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/tmdb"
)

// Services groups all application services needed by the HTTP layer.
type Services struct {
	Auth    *service.AuthService
	Movies  *service.MovieService
	Stats   *service.StatsService
	Backups *service.BackupService
	TMDB    *ExternalTMDB
}

// ExternalTMDB wraps the TMDB client for HTTP handlers.
type ExternalTMDB struct {
	Client *tmdb.Client
}

func (e *ExternalTMDB) SearchMovies(r *http.Request, query string, year int) ([]tmdb.SearchResult, error) {
	if e == nil || e.Client == nil {
		return nil, nil
	}
	return e.Client.SearchMovies(r.Context(), query, year)
}

// NewRouter builds the complete HTTP mux with all routes and middleware.
// Public auth routes: 5 req/s per IP, burst 20.
// Protected API routes: same rate limit, all behind JWT middleware.
func NewRouter(svcs Services, jwtSecret []byte) http.Handler {
	mux := http.NewServeMux()
	rl := RateLimiter(rate.Limit(5), 20)
	auth := func(h http.Handler) http.Handler { return rl(JWTMiddleware(jwtSecret, h)) }

	mux.HandleFunc("GET /health", healthHandler)

	// — Public auth routes —
	ah := &authHandler{auth: svcs.Auth}
	mux.Handle("POST /api/register", rl(http.HandlerFunc(ah.register)))
	mux.Handle("POST /api/login", rl(http.HandlerFunc(ah.login)))
	mux.Handle("POST /api/refresh", rl(http.HandlerFunc(ah.refresh)))
	mux.Handle("GET /api/me", auth(http.HandlerFunc(ah.me)))

	// — Protected movie routes —
	if svcs.Movies != nil {
		mh := &movieHandler{movies: svcs.Movies}
		mux.Handle("GET /api/v1/movies", auth(http.HandlerFunc(mh.list)))
		mux.Handle("POST /api/v1/movies", auth(http.HandlerFunc(mh.create)))
		mux.Handle("GET /api/v1/movies/{id}", auth(http.HandlerFunc(mh.get)))
		mux.Handle("PUT /api/v1/movies/{id}", auth(http.HandlerFunc(mh.update)))
		mux.Handle("DELETE /api/v1/movies/{id}", auth(http.HandlerFunc(mh.delete)))
		mux.Handle("PUT /api/v1/movies/{id}/watch", auth(http.HandlerFunc(mh.watch)))
	}

	// — Stats —
	if svcs.Stats != nil {
		sh := &statsHandler{stats: svcs.Stats}
		mux.Handle("GET /api/v1/stats", auth(http.HandlerFunc(sh.get)))
	}

	// — Sync —
	if svcs.Movies != nil {
		syh := &syncHandler{movies: svcs.Movies}
		mux.Handle("GET /api/v1/sync", auth(http.HandlerFunc(syh.export)))
		mux.Handle("POST /api/v1/sync", auth(http.HandlerFunc(syh.importData)))
	}

	// — External search (TMDB proxy) —
	if svcs.TMDB != nil && svcs.TMDB.Client != nil {
		eh := &externalHandler{tmdb: svcs.TMDB}
		mux.Handle("GET /api/v1/search/external", auth(http.HandlerFunc(eh.search)))
	}

	// — Backup config/state —
	if svcs.Backups != nil {
		bh := &backupHandler{backups: svcs.Backups}
		mux.Handle("GET /api/v1/backup/config", auth(http.HandlerFunc(bh.exportConfig)))
		mux.Handle("PUT /api/v1/backup/config", auth(http.HandlerFunc(bh.importConfig)))
		mux.Handle("GET /api/v1/backup/state", auth(http.HandlerFunc(bh.exportState)))
		mux.Handle("PUT /api/v1/backup/state", auth(http.HandlerFunc(bh.importState)))
		mux.Handle("GET /api/v1/backup", auth(http.HandlerFunc(bh.exportSnapshot)))
		mux.Handle("PUT /api/v1/backup", auth(http.HandlerFunc(bh.importSnapshot)))
	}

	return mux
}

