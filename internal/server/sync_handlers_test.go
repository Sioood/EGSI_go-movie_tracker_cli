package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtectedSyncRequiresAuth(t *testing.T) {
	router := newFullRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestSyncImportRejectsInvalidMovie(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "sync-invalid")

	rr := authPost(t, router, "/api/v1/sync", token, map[string]any{
		"movies": []map[string]any{
			{"id": "movie-invalid", "title": "", "year": 2020},
		},
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 for invalid movie payload, got %d: %s", rr.Code, rr.Body)
	}
}

func TestSyncImportRejectsInvalidJSON(t *testing.T) {
	router := newFullRouter(t)
	token := registerAndLogin(t, router, "sync-json")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for invalid JSON, got %d", rr.Code)
	}
}
