package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestMovieRepositoryCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	watchEntries := NewWatchEntryRepository(db)

	created, err := movies.Create(ctx, domain.Movie{
		UserID:     "user-1",
		Title:      "Arrival",
		Year:       2016,
		ExternalID: "tmdb:329865",
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated movie id")
	}

	got, err := movies.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get movie: %v", err)
	}
	if got.Title != "Arrival" || got.UserID != "user-1" || got.Year != 2016 {
		t.Fatalf("unexpected movie: %+v", got)
	}

	_, err = movies.Create(ctx, domain.Movie{UserID: "user-2", Title: "Heat", Year: 1995})
	if err != nil {
		t.Fatalf("create second user movie: %v", err)
	}

	list, err := movies.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list movies: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected one movie for user-1, got %+v", list)
	}

	created.Title = "Arrival Updated"
	created.Year = 2017
	updated, err := movies.Update(ctx, created)
	if err != nil {
		t.Fatalf("update movie: %v", err)
	}
	if updated.Title != "Arrival Updated" || updated.Year != 2017 {
		t.Fatalf("unexpected updated movie: %+v", updated)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) && !updated.UpdatedAt.Equal(updated.CreatedAt) {
		t.Fatalf("updated_at should not be before created_at: %+v", updated)
	}

	rating := 8.5
	watchedAt := time.Date(2026, 7, 4, 20, 30, 0, 0, time.UTC)
	entry, err := watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID:     created.ID,
		Watched:     true,
		Rating:      &rating,
		RatingScale: 10,
		Review:      "Smart and quiet sci-fi.",
		WatchedAt:   &watchedAt,
	})
	if err != nil {
		t.Fatalf("upsert watch entry: %v", err)
	}
	if entry.ID == "" || !entry.Watched || entry.Rating == nil || *entry.Rating != rating {
		t.Fatalf("unexpected watch entry: %+v", entry)
	}

	rating = 9
	entry.Rating = &rating
	entry.Review = "Even better on rewatch."
	updatedEntry, err := watchEntries.Upsert(ctx, entry)
	if err != nil {
		t.Fatalf("update watch entry: %v", err)
	}
	if updatedEntry.ID != entry.ID || updatedEntry.Rating == nil || *updatedEntry.Rating != 9 {
		t.Fatalf("unexpected updated watch entry: %+v", updatedEntry)
	}

	if err := movies.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete movie: %v", err)
	}

	_, err = movies.GetByID(ctx, created.ID)
	if !errors.Is(err, apperrors.ErrMovieNotFound) {
		t.Fatalf("expected ErrMovieNotFound, got %v", err)
	}

	_, err = watchEntries.GetByMovieID(ctx, created.ID)
	if !errors.Is(err, apperrors.ErrWatchEntryNotFound) {
		t.Fatalf("expected cascaded watch entry delete, got %v", err)
	}
}

func TestMovieRepositorySearch(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	movies := NewMovieRepository(db)
	watchEntries := NewWatchEntryRepository(db)

	arrival, err := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "Arrival", Year: 2016})
	if err != nil {
		t.Fatalf("create arrival: %v", err)
	}
	heat, err := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "Heat", Year: 1995})
	if err != nil {
		t.Fatalf("create heat: %v", err)
	}
	matrix, err := movies.Create(ctx, domain.Movie{UserID: "user-1", Title: "The Matrix", Year: 1999})
	if err != nil {
		t.Fatalf("create matrix: %v", err)
	}
	_, err = movies.Create(ctx, domain.Movie{UserID: "user-2", Title: "Arrival", Year: 2016})
	if err != nil {
		t.Fatalf("create other user movie: %v", err)
	}

	arrivalRating := 8.5
	arrivalDate := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	_, err = watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID:     arrival.ID,
		Watched:     true,
		Rating:      &arrivalRating,
		RatingScale: 10,
		WatchedAt:   &arrivalDate,
	})
	if err != nil {
		t.Fatalf("watch arrival: %v", err)
	}

	heatRating := 9.5
	heatDate := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	_, err = watchEntries.Upsert(ctx, domain.WatchEntry{
		MovieID:     heat.ID,
		Watched:     true,
		Rating:      &heatRating,
		RatingScale: 10,
		WatchedAt:   &heatDate,
	})
	if err != nil {
		t.Fatalf("watch heat: %v", err)
	}

	result, err := movies.Search(ctx, domain.MovieSearchParams{
		UserID: "user-1",
		Query:  "arr",
		Filter: domain.MovieFilterAll,
		Sort:   domain.MovieSortTitle,
	})
	if err != nil {
		t.Fatalf("search by title: %v", err)
	}
	assertMovieTitles(t, result, "Arrival")

	result, err = movies.Search(ctx, domain.MovieSearchParams{UserID: "user-1", Filter: domain.MovieFilterWatched, Sort: domain.MovieSortTitle})
	if err != nil {
		t.Fatalf("filter watched: %v", err)
	}
	assertMovieTitles(t, result, "Arrival", "Heat")

	result, err = movies.Search(ctx, domain.MovieSearchParams{UserID: "user-1", Filter: domain.MovieFilterUnwatched, Sort: domain.MovieSortTitle})
	if err != nil {
		t.Fatalf("filter unwatched: %v", err)
	}
	assertMovieTitles(t, result, "The Matrix")

	result, err = movies.Search(ctx, domain.MovieSearchParams{UserID: "user-1", Filter: domain.MovieFilterRated, Sort: domain.MovieSortRating})
	if err != nil {
		t.Fatalf("filter rated: %v", err)
	}
	assertMovieTitles(t, result, "Heat", "Arrival")

	result, err = movies.Search(ctx, domain.MovieSearchParams{UserID: "user-1", Filter: domain.MovieFilterAll, Sort: domain.MovieSortDate})
	if err != nil {
		t.Fatalf("sort date: %v", err)
	}
	assertMovieTitles(t, result, "Heat", "Arrival", "The Matrix")

	if matrix.ID == "" {
		t.Fatal("expected matrix id")
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.OpenAndMigrate(
		"file:movietracker_repository_test?mode=memory&cache=shared&_pragma=foreign_keys(1)",
		database.ClientMigrations,
		"migrations/client",
	)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	return db
}

func assertMovieTitles(t *testing.T, movies []domain.Movie, titles ...string) {
	t.Helper()

	if len(movies) != len(titles) {
		t.Fatalf("expected %d movies, got %d: %+v", len(titles), len(movies), movies)
	}
	for index, title := range titles {
		if movies[index].Title != title {
			t.Fatalf("movie %d: expected %q, got %q", index, title, movies[index].Title)
		}
	}
}
