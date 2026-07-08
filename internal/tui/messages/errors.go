package messages

import (
	"errors"
	"fmt"
	"strings"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

// UserMessage maps application errors to French user-facing text.
func UserMessage(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, apperrors.ErrMovieNotFound):
		return "Film introuvable."
	case errors.Is(err, apperrors.ErrWatchEntryNotFound):
		return "Entrée de visionnage introuvable."
	case errors.Is(err, apperrors.ErrUserNotFound):
		return "Utilisateur introuvable."
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		return "Identifiants invalides."
	case errors.Is(err, apperrors.ErrEmailAlreadyExists):
		return "Email déjà utilisé."
	case errors.Is(err, apperrors.ErrUnauthorized):
		return "Session expirée. Reconnecte-toi."
	case errors.Is(err, apperrors.ErrForbidden):
		return "Action non autorisée."
	case errors.Is(err, apperrors.ErrNetwork):
		return "Erreur réseau. Vérifie la connexion et l'URL du serveur."
	case errors.Is(err, apperrors.ErrDB):
		return "Erreur de base de données."
	case errors.Is(err, apperrors.ErrConfigMissing):
		return "Configuration manquante."
	case errors.Is(err, apperrors.ErrInvalidRating):
		return mapInvalidRating(err)
	case errors.Is(err, apperrors.ErrValidation):
		return mapValidation(err)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "année invalide"):
		return "Année invalide."
	case strings.Contains(msg, "note invalide"):
		return "Note invalide."
	case strings.Contains(msg, "date invalide"):
		return "Date invalide, format attendu AAAA-MM-JJ."
	case strings.Contains(msg, "l'email est requis"):
		return "L'email est requis."
	case strings.Contains(msg, "mot de passe doit contenir"):
		return "Le mot de passe doit contenir au moins 8 caractères."
	case strings.Contains(msg, "service films indisponible"):
		return UI.MovieServiceUnavailable
	case strings.Contains(msg, "aucun film sélectionné"):
		return UI.NoMovieSelected
	}

	return "Une erreur inattendue s'est produite."
}

func mapValidation(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "title is required"):
		return "Le titre est obligatoire."
	case strings.Contains(msg, "user id is required"):
		return "Identifiant utilisateur requis."
	case strings.Contains(msg, "year cannot be negative"):
		return "L'année ne peut pas être négative."
	case strings.Contains(msg, "movie id is required"):
		return "Identifiant film requis."
	case strings.Contains(msg, "email invalide"):
		return "Email invalide."
	case strings.Contains(msg, "mot de passe trop court"):
		return "Le mot de passe doit contenir au moins 8 caractères."
	case strings.Contains(msg, "movie belongs to another user"):
		return "Ce film appartient à un autre utilisateur."
	default:
		return "Données invalides."
	}
}

func mapInvalidRating(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "rating must be between") {
		return "La note doit être comprise entre 0 et 10."
	}
	if strings.Contains(msg, "rating scale must be positive") {
		return "Échelle de notation invalide."
	}
	return "Note invalide."
}

// Prefix formats an error with a French prefix.
func Prefix(prefix string, err error) string {
	return fmt.Sprintf("%s %s", prefix, UserMessage(err))
}
