package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/logging"
)

var httpLog = logging.New("http")

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		httpLog.Error("encode JSON response", slog.Any("err", err))
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeInternalError(w http.ResponseWriter, msg string, err error) {
	if err != nil {
		httpLog.Error(msg, slog.Any("err", err))
	}
	writeError(w, http.StatusInternalServerError, "erreur interne")
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
		writeInternalError(w, "auth handler", err)
	}
}

func writeMovieError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrMovieNotFound):
		writeError(w, http.StatusNotFound, "film introuvable")
	case errors.Is(err, apperrors.ErrForbidden):
		writeError(w, http.StatusForbidden, "accès interdit")
	case errors.Is(err, apperrors.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, apperrors.ErrInvalidRating):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeInternalError(w, "movie handler", err)
	}
}
