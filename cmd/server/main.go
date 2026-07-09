package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/logging"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/server"
	"github.com/movietracker/movie-tracker/internal/service"
	"github.com/movietracker/movie-tracker/internal/tmdb"
	"github.com/movietracker/movie-tracker/internal/version"
)

func main() {
	logger := logging.New("server")

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		logger.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	if err := os.MkdirAll("data", 0o755); err != nil {
		logger.Error("create data directory", "err", err)
		os.Exit(1)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/server.db"
	}
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Error("create database directory", "dir", dir, "err", err)
			os.Exit(1)
		}
	}
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)"
	db, err := database.OpenAndMigrate(dsn, database.ServerMigrations, "migrations/server")
	if err != nil {
		logger.Error("database migration failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)
	watchEntryRepo := repository.NewWatchEntryRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	backupRepo := repository.NewBackupRepository(db)

	authSvc := service.NewAuthService(userRepo, jwtSecret)
	movieSvc := service.NewMovieService(movieRepo, watchEntryRepo)
	statsSvc := service.NewStatsService(statsRepo)
	backupSvc := service.NewBackupService(backupRepo)

	var externalTMDB *server.ExternalTMDB
	if apiKey := os.Getenv("TMDB_API_KEY"); apiKey != "" {
		externalTMDB = &server.ExternalTMDB{Client: tmdb.NewClient(apiKey)}
	} else {
		logger.Warn("TMDB_API_KEY not set, external search disabled")
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		} else {
			addr = ":8080"
		}
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("listen", "addr", addr, "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Handler: server.NewRouter(server.Services{
			Auth:    authSvc,
			Movies:  movieSvc,
			Stats:   statsSvc,
			Backups: backupSvc,
			TMDB:    externalTMDB,
		}, jwtSecret),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("MovieTracker server ready", "version", version.Version, "addr", addr, "database", dbPath)

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "err", err)
	}
	logger.Info("server stopped")
}
