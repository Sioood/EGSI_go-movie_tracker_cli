package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/service"
)

type authHandler struct {
	auth *service.AuthService
}

// POST /api/register
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	pair, err := h.auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, pair)
}

// POST /api/login
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	pair, err := h.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// POST /api/refresh
func (h *authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	pair, err := h.auth.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// GET /api/me  (requires JWT)
func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "non authentifié")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"id":    claims.UserID,
		"email": claims.Email,
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, apperrors.ErrEmailAlreadyExists):
		writeError(w, http.StatusConflict, "email déjà utilisé")
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "identifiants invalides")
	default:
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
