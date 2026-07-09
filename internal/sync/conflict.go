package sync

import (
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
)

func moviesConflict(local, remote domain.Movie) bool {
	return local.Title != remote.Title ||
		local.Year != remote.Year ||
		local.ExternalID != remote.ExternalID
}

func watchEntriesConflict(local, remote domain.WatchEntry) bool {
	if local.Watched != remote.Watched ||
		local.RatingScale != remote.RatingScale ||
		local.Review != remote.Review {
		return true
	}
	if !floatPtrEqual(local.Rating, remote.Rating) {
		return true
	}
	return !timePtrEqual(local.WatchedAt, remote.WatchedAt)
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.UTC().Equal(b.UTC())
}
