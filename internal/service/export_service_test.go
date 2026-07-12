package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/domain"
)

func TestExportServiceJSONAndCSV(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rating := 8.5
	watchedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	movies := NewMovieService(exportMovieStore{
		movies: []domain.Movie{{
			ID:         "movie-1",
			UserID:     "user-1",
			Title:      "Inception",
			Year:       2010,
			ExternalID: "tmdb:27205",
		}},
	}, exportWatchStore{
		entries: map[string]domain.WatchEntry{
			"movie-1": {
				ID:          "entry-1",
				MovieID:     "movie-1",
				Watched:     true,
				Rating:      &rating,
				RatingScale: 10,
				Review:      "Excellent",
				WatchedAt:   &watchedAt,
				UpdatedAt:   watchedAt,
			},
		},
	})
	exportSvc := NewExportService(movies)

	jsonPath, err := exportSvc.ExportJSON(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("export json: %v", err)
	}
	if !strings.HasSuffix(jsonPath, ".json") {
		t.Fatalf("unexpected json path: %s", jsonPath)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var payload MovieExport
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if len(payload.Movies) != 1 || payload.Movies[0].Title != "Inception" {
		t.Fatalf("unexpected movies: %+v", payload.Movies)
	}
	if len(payload.WatchEntries) != 1 || payload.WatchEntries[0].Review != "Excellent" {
		t.Fatalf("unexpected watch entries: %+v", payload.WatchEntries)
	}

	csvPath, err := exportSvc.ExportCSV(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if filepath.Dir(csvPath) != filepath.Dir(jsonPath) {
		t.Fatalf("csv and json should share export dir")
	}

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("open csv: %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + row, got %d rows", len(records))
	}
	if records[1][0] != "Inception" || records[1][2] != "tmdb:27205" || records[1][3] != "true" {
		t.Fatalf("unexpected csv row: %+v", records[1])
	}
}

type exportMovieStore struct {
	movies []domain.Movie
}

func (s exportMovieStore) Create(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	return movie, nil
}

func (s exportMovieStore) GetByID(ctx context.Context, id string) (domain.Movie, error) {
	for _, movie := range s.movies {
		if movie.ID == id {
			return movie, nil
		}
	}
	return domain.Movie{}, nil
}

func (s exportMovieStore) GetByExternalID(ctx context.Context, userID, externalID string) (domain.Movie, error) {
	for _, movie := range s.movies {
		if movie.UserID == userID && movie.ExternalID == externalID {
			return movie, nil
		}
	}
	return domain.Movie{}, apperrors.ErrMovieNotFound
}

func (s exportMovieStore) ListByUser(ctx context.Context, userID string) ([]domain.Movie, error) {
	var result []domain.Movie
	for _, movie := range s.movies {
		if movie.UserID == userID {
			result = append(result, movie)
		}
	}
	return result, nil
}

func (s exportMovieStore) Search(ctx context.Context, params domain.MovieSearchParams) ([]domain.Movie, error) {
	return s.ListByUser(ctx, params.UserID)
}

func (s exportMovieStore) Update(ctx context.Context, movie domain.Movie) (domain.Movie, error) {
	return movie, nil
}

func (s exportMovieStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (s exportMovieStore) SyncUpsert(ctx context.Context, movie domain.Movie) (domain.Movie, bool, error) {
	return movie, true, nil
}

type exportWatchStore struct {
	entries map[string]domain.WatchEntry
}

func (s exportWatchStore) Upsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, error) {
	return entry, nil
}

func (s exportWatchStore) GetByMovieID(ctx context.Context, movieID string) (domain.WatchEntry, error) {
	return s.entries[movieID], nil
}

func (s exportWatchStore) ListByMovieIDs(ctx context.Context, movieIDs []string) ([]domain.WatchEntry, error) {
	entries := make([]domain.WatchEntry, 0, len(movieIDs))
	for _, movieID := range movieIDs {
		if entry, ok := s.entries[movieID]; ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s exportWatchStore) DeleteByMovieID(ctx context.Context, movieID string) error {
	return nil
}

func (s exportWatchStore) SyncUpsert(ctx context.Context, entry domain.WatchEntry) (domain.WatchEntry, bool, error) {
	return entry, true, nil
}
