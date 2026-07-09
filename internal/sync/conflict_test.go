package sync

import (
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestMoviesConflictDetectsDivergence(t *testing.T) {
	local := domain.Movie{Title: "Inception", Year: 2010}
	remote := domain.Movie{Title: "Inception", Year: 2011}
	if !moviesConflict(local, remote) {
		t.Fatal("expected movie conflict")
	}
}

func TestWatchEntriesConflictDetectsRatingChange(t *testing.T) {
	localRating := 8.0
	remoteRating := 9.0
	local := domain.WatchEntry{Rating: &localRating, RatingScale: 10}
	remote := domain.WatchEntry{Rating: &remoteRating, RatingScale: 10}
	if !watchEntriesConflict(local, remote) {
		t.Fatal("expected watch entry conflict")
	}
}

func TestTimePtrEqual(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(time.Hour)
	if !timePtrEqual(&now, &now) {
		t.Fatal("expected equal timestamps")
	}
	if timePtrEqual(&now, &later) {
		t.Fatal("expected different timestamps")
	}
}
