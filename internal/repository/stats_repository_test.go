package repository

import (
	"context"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestStatsRepository(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	watchEntries := NewWatchEntryRepository(db)
	stats := NewStatsRepository(db)

	// Empty state
	s, err := stats.GetStats(ctx, "user-stats")
	if err != nil {
		t.Fatalf("get empty stats: %v", err)
	}
	if s.TotalMovies != 0 || s.TotalWatched != 0 || s.TotalRated != 0 {
		t.Fatalf("expected zero stats, got %+v", s)
	}

	// Add two movies
	m1, _ := movies.Create(ctx, domain.Movie{UserID: "user-stats", Title: "Arrival", Year: 2016})
	m2, _ := movies.Create(ctx, domain.Movie{UserID: "user-stats", Title: "Heat", Year: 1995})
	m3, _ := movies.Create(ctx, domain.Movie{UserID: "user-stats", Title: "Inception", Year: 2010})

	// Watch m1 and m2 with ratings
	r1 := 9.0
	watchedAt1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID: m1.ID, Watched: true, Rating: &r1, RatingScale: 10, WatchedAt: &watchedAt1,
	})

	r2 := 6.0
	watchedAt2 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID: m2.ID, Watched: true, Rating: &r2, RatingScale: 10, WatchedAt: &watchedAt2,
	})

	// m3: tracked but not watched
	watchEntries.Upsert(ctx, domain.WatchEntry{MovieID: m3.ID, Watched: false})

	s, err = stats.GetStats(ctx, "user-stats")
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	if s.TotalMovies != 3 {
		t.Errorf("TotalMovies: want 3, got %d", s.TotalMovies)
	}
	if s.TotalWatched != 2 {
		t.Errorf("TotalWatched: want 2, got %d", s.TotalWatched)
	}
	if s.TotalRated != 2 {
		t.Errorf("TotalRated: want 2, got %d", s.TotalRated)
	}

	wantAvg := (9.0 + 6.0) / 2.0
	if s.AverageRating != wantAvg {
		t.Errorf("AverageRating: want %.1f, got %.1f", wantAvg, s.AverageRating)
	}

	if len(s.BestMovies) == 0 || s.BestMovies[0].Movie.Title != "Arrival" {
		t.Errorf("BestMovies[0]: want Arrival, got %+v", s.BestMovies)
	}
	if len(s.WorstMovies) == 0 || s.WorstMovies[0].Movie.Title != "Heat" {
		t.Errorf("WorstMovies[0]: want Heat, got %+v", s.WorstMovies)
	}

	// Histogram: expect two buckets (March and May 2026)
	if len(s.ByMonth) != 2 {
		t.Fatalf("ByMonth: want 2 buckets, got %d: %+v", len(s.ByMonth), s.ByMonth)
	}
	if s.ByMonth[0].Month != 3 || s.ByMonth[0].Count != 1 {
		t.Errorf("ByMonth[0]: want March 1, got %+v", s.ByMonth[0])
	}
	if s.ByMonth[1].Month != 5 || s.ByMonth[1].Count != 1 {
		t.Errorf("ByMonth[1]: want May 1, got %+v", s.ByMonth[1])
	}

	// Isolation: different user should see zero stats
	s2, err := stats.GetStats(ctx, "other-user")
	if err != nil {
		t.Fatalf("get other user stats: %v", err)
	}
	if s2.TotalMovies != 0 {
		t.Errorf("other user TotalMovies: want 0, got %d", s2.TotalMovies)
	}

	_ = domain.MonthBucket{Year: 2026, Month: time.March, Count: 1}
}
