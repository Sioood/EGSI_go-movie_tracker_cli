package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/domain"
)

const (
	exportDirPerm  = 0o700
	exportFilePerm = 0o600
)

// MovieExport is the JSON export payload for a user's movie collection.
type MovieExport struct {
	Movies       []domain.Movie      `json:"movies"`
	WatchEntries []domain.WatchEntry `json:"watch_entries"`
	ExportedAt   time.Time           `json:"exported_at"`
}

// ExportService exports movies and watch entries to local files.
type ExportService struct {
	movies *MovieService
}

// NewExportService creates an ExportService.
func NewExportService(movies *MovieService) *ExportService {
	return &ExportService{movies: movies}
}

// ExportJSON writes the user's collection as JSON under ~/.config/movietracker/exports/.
func (s *ExportService) ExportJSON(ctx context.Context, userID string) (string, error) {
	payload, err := s.buildExport(ctx, userID)
	if err != nil {
		return "", err
	}

	dir, err := exportDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, fmt.Sprintf("movies-%s.json", timestamp()))
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export: %w", err)
	}
	if err := os.WriteFile(path, data, exportFilePerm); err != nil {
		return "", fmt.Errorf("write export: %w", err)
	}
	return path, nil
}

// ExportCSV writes the user's collection as CSV under ~/.config/movietracker/exports/.
func (s *ExportService) ExportCSV(ctx context.Context, userID string) (string, error) {
	payload, err := s.buildExport(ctx, userID)
	if err != nil {
		return "", err
	}

	dir, err := exportDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, fmt.Sprintf("movies-%s.csv", timestamp()))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, exportFilePerm)
	if err != nil {
		return "", fmt.Errorf("create csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{
		"title", "year", "external_id", "watched", "rating", "rating_scale", "review", "watched_at", "updated_at",
	}); err != nil {
		return "", fmt.Errorf("write csv header: %w", err)
	}

	entryByMovie := make(map[string]domain.WatchEntry, len(payload.WatchEntries))
	for _, entry := range payload.WatchEntries {
		entryByMovie[entry.MovieID] = entry
	}

	for _, movie := range payload.Movies {
		entry := entryByMovie[movie.ID]
		row, err := movieCSVRow(movie, entry)
		if err != nil {
			return "", err
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush csv: %w", err)
	}
	return path, nil
}

func (s *ExportService) buildExport(ctx context.Context, userID string) (MovieExport, error) {
	if userID == "" {
		return MovieExport{}, fmt.Errorf("%w: user id is required", apperrors.ErrValidation)
	}

	movies, err := s.movies.ListMovies(ctx, userID)
	if err != nil {
		return MovieExport{}, err
	}

	entries := make([]domain.WatchEntry, 0, len(movies))
	for _, movie := range movies {
		entry, err := s.movies.GetWatchEntry(ctx, movie.ID)
		if errors.Is(err, apperrors.ErrWatchEntryNotFound) {
			continue
		}
		if err != nil {
			return MovieExport{}, err
		}
		entries = append(entries, entry)
	}

	return MovieExport{
		Movies:       movies,
		WatchEntries: entries,
		ExportedAt:   time.Now().UTC(),
	}, nil
}

func movieCSVRow(movie domain.Movie, entry domain.WatchEntry) ([]string, error) {
	watched := "false"
	if entry.Watched {
		watched = "true"
	}

	rating := ""
	if entry.Rating != nil {
		rating = strconv.FormatFloat(*entry.Rating, 'f', -1, 64)
	}

	ratingScale := ""
	if entry.MovieID != "" {
		ratingScale = strconv.Itoa(entry.RatingScale)
	}

	watchedAt := ""
	if entry.WatchedAt != nil {
		watchedAt = entry.WatchedAt.UTC().Format(time.RFC3339)
	}

	updatedAt := ""
	if !entry.UpdatedAt.IsZero() {
		updatedAt = entry.UpdatedAt.UTC().Format(time.RFC3339)
	}

	return []string{
		movie.Title,
		strconv.Itoa(movie.Year),
		movie.ExternalID,
		watched,
		rating,
		ratingScale,
		entry.Review,
		watchedAt,
		updatedAt,
	}, nil
}

func exportDir() (string, error) {
	base, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "exports")
	if err := os.MkdirAll(dir, exportDirPerm); err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}
	return dir, nil
}

func timestamp() string {
	return time.Now().UTC().Format("2006-01-02-150405")
}
