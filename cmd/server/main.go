package main

import (
	"os"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/logging"
)

func main() {
	logger := logging.New("server")

	if err := os.MkdirAll("data", 0o755); err != nil {
		logger.Error("create data directory", "err", err)
		os.Exit(1)
	}

	dbPath := "data/server.db"
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ServerMigrations, "migrations/server")
	if err != nil {
		logger.Error("database migration failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("MovieTracker server ready", "phase", 0, "database", dbPath)
}
