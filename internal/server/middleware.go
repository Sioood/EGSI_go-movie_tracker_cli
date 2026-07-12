package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/movietracker/movie-tracker/internal/auth"
)

// JWTMiddleware validates the Bearer token in the Authorization header and
// injects the parsed claims into the request context.
func JWTMiddleware(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "token manquant")
			return
		}

		claims, err := auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token invalide")
			return
		}

		next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
	})
}

// SecurityHeaders adds baseline HTTP security headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter returns a per-IP rate-limiting middleware.
// Each unique IP is allowed up to burst requests immediately, then rps per second.
// When trustedProxy is false, X-Forwarded-For is ignored to prevent spoofing.
func RateLimiter(rps rate.Limit, burst int, trustedProxy bool) func(http.Handler) http.Handler {
	type entry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var mu sync.Mutex
	limiters := make(map[string]*entry)

	// Purge stale entries every minute to prevent unbounded memory growth.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, e := range limiters {
				if time.Since(e.lastSeen) > 5*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		e, ok := limiters[ip]
		if !ok {
			e = &entry{limiter: rate.NewLimiter(rps, burst)}
			limiters[ip] = e
		}
		e.lastSeen = time.Now()
		return e.limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !getLimiter(clientIP(r, trustedProxy)).Allow() {
				writeError(w, http.StatusTooManyRequests, "trop de requêtes")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request, trustedProxy bool) string {
	if trustedProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx >= 0 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		addr = addr[:idx]
	}
	return addr
}
