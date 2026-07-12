package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/movietracker/movie-tracker/internal/server"
)

func TestSecurityHeaders(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options: want nosniff, got %q", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options: want DENY, got %q", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy: want strict-origin-when-cross-origin, got %q", got)
	}
}

func TestTrustedProxyUsesXForwardedFor(t *testing.T) {
	router := server.NewRouter(server.Services{}, []byte("phase-6-test-secret-key"), server.RouterOptions{TrustedProxy: true})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "203.0.113.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 203.0.113.1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("health: want 200, got %d", rr.Code)
	}
}

func TestUntrustedProxyIgnoresXForwardedForForRateLimit(t *testing.T) {
	router := server.NewRouter(server.Services{}, []byte("phase-6-test-secret-key"))

	var got429 bool
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i%200+1))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("rate limit should apply to RemoteAddr when TRUSTED_PROXY is false, even with varying X-Forwarded-For")
	}
}

func TestTrustedProxyUsesXForwardedForForRateLimit(t *testing.T) {
	router := server.NewRouter(server.Services{}, []byte("phase-6-test-secret-key"), server.RouterOptions{TrustedProxy: true})

	var got429 bool
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i+1))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if got429 {
		t.Fatal("each distinct X-Forwarded-For should have its own rate limit bucket when TRUSTED_PROXY is true")
	}
}

func TestJWTMiddlewareMissingBearerPrefix(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Token not-a-bearer")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestJWTMiddlewareEmptyBearerToken(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestJWTMiddlewareMalformedToken(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer not.valid.jwt")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}
