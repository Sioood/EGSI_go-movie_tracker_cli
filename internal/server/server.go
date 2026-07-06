package server

import (
	"net/http"

	"golang.org/x/time/rate"

	"github.com/movietracker/movie-tracker/internal/service"
)

// NewRouter builds the HTTP mux with all auth routes and middleware applied.
// Rate limit: 5 req/s per IP, burst of 20.
func NewRouter(authSvc *service.AuthService, jwtSecret []byte) http.Handler {
	mux := http.NewServeMux()
	h := &authHandler{auth: authSvc}
	rl := RateLimiter(rate.Limit(5), 20)

	mux.Handle("POST /api/register", rl(http.HandlerFunc(h.register)))
	mux.Handle("POST /api/login", rl(http.HandlerFunc(h.login)))
	mux.Handle("POST /api/refresh", rl(http.HandlerFunc(h.refresh)))
	mux.Handle("GET /api/me", rl(JWTMiddleware(jwtSecret, http.HandlerFunc(h.me))))

	return mux
}
