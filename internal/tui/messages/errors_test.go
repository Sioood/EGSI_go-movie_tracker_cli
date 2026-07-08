package messages_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/tui/messages"
)

func TestUserMessageKnownErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{apperrors.ErrMovieNotFound, "Film introuvable."},
		{apperrors.ErrWatchEntryNotFound, "Entrée de visionnage introuvable."},
		{apperrors.ErrInvalidCredentials, "Identifiants invalides."},
		{apperrors.ErrEmailAlreadyExists, "Email déjà utilisé."},
		{apperrors.ErrUnauthorized, "Session expirée. Reconnecte-toi."},
		{apperrors.ErrForbidden, "Action non autorisée."},
		{apperrors.ErrNetwork, "Erreur réseau. Vérifie la connexion et l'URL du serveur."},
		{apperrors.ErrDB, "Erreur de base de données."},
		{fmt.Errorf("%w: title is required", apperrors.ErrValidation), "Le titre est obligatoire."},
		{fmt.Errorf("%w: user id is required", apperrors.ErrValidation), "Identifiant utilisateur requis."},
		{fmt.Errorf("%w: rating must be between 0 and 10", apperrors.ErrInvalidRating), "La note doit être comprise entre 0 et 10."},
		{errors.New("année invalide"), "Année invalide."},
		{errors.New("date invalide, format attendu YYYY-MM-DD"), "Date invalide, format attendu AAAA-MM-JJ."},
	}

	for _, tc := range tests {
		got := messages.UserMessage(tc.err)
		if got != tc.want {
			t.Errorf("UserMessage(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestUserMessageNil(t *testing.T) {
	if got := messages.UserMessage(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFilterAndSortLabels(t *testing.T) {
	if messages.FilterLabel(domain.MovieFilterAll) != "tous" {
		t.Fatal("expected default filter label")
	}
	if messages.SortLabel(domain.MovieSortTitle) != "titre" {
		t.Fatal("expected default sort label")
	}
}
