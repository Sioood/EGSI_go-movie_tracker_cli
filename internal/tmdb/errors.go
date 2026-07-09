package tmdb

import "errors"

// ErrUnavailable indicates TMDB search cannot run with the current configuration.
var ErrUnavailable = errors.New("recherche TMDB indisponible")
