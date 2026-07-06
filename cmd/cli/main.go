package main

import (
	"os"
	"path/filepath"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/logging"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// localService combines MovieService and StatsService to satisfy tui.MovieClient.
type localService struct {
	*service.MovieService
	*service.StatsService
}

func main() {
	logger := logging.New("cli")

	dbPath := filepath.Join("data", "client.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		logger.Error("create data directory", "err", err)
		os.Exit(1)
	}

	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ClientMigrations, "migrations/client")
	if err != nil {
		logger.Error("database migration failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	movieRepository := repository.NewMovieRepository(db)
	watchEntryRepository := repository.NewWatchEntryRepository(db)
	statsRepository := repository.NewStatsRepository(db)

	movieService := service.NewMovieService(movieRepository, watchEntryRepository)
	statsService := service.NewStatsService(statsRepository)

	svc := &localService{movieService, statsService}

	logger.Info("MovieTracker CLI ready", "phase", 5, "database", dbPath)

	program := tea.NewProgram(tui.New(svc), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		logger.Error("tui failed", "err", err)
		os.Exit(1)
	}
}
