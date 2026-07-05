package main

import (
	"os"
	"path/filepath"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/logging"
)

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

	logger.Info("MovieTracker CLI ready", "phase", 1, "database", dbPath)
}
