package apperrors

import "errors"

var (
	ErrMovieNotFound      = errors.New("movie not found")
	ErrWatchEntryNotFound = errors.New("watch entry not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation error")
	ErrNetwork            = errors.New("network error")
	ErrDB                 = errors.New("database error")
	ErrConfigMissing      = errors.New("config missing")
	ErrInvalidRating      = errors.New("invalid rating")
)
