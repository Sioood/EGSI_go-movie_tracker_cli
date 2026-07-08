package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/config"
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

// authAdapter bridges internal/client.AuthClient to tui.AuthClient.
type authAdapter struct {
	*client.AuthClient
}

func (a *authAdapter) Me(ctx context.Context, accessToken string) (tui.UserInfo, error) {
	info, err := a.AuthClient.Me(ctx, accessToken)
	if err != nil {
		return tui.UserInfo{}, err
	}
	return tui.UserInfo{ID: info.ID, Email: info.Email}, nil
}

func main() {
	logger := logging.New("cli")

	if _, err := config.Dir(); err != nil {
		logger.Error("config directory", "err", err)
		os.Exit(1)
	}

	appCfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	sess, err := config.LoadSession()
	if err != nil {
		logger.Error("load session", "err", err)
		os.Exit(1)
	}

	authClient := client.NewAuthClient(appCfg.ServerURL)
	sessionState := tuiSessionFromConfig(sess)

	if !appCfg.OfflineMode && sess.RefreshToken != "" {
		restored, err := client.RestoreSession(context.Background(), authClient, sess)
		if err != nil {
			logger.Warn("session restore failed, clearing tokens", "err", err)
			if clearErr := config.ClearSession(); clearErr != nil {
				logger.Error("clear session", "err", clearErr)
			}
			sessionState = tui.SessionState{}
		} else {
			sess = restored.Session
			sessionState = tuiSessionFromConfig(sess)
			if saveErr := config.SaveSession(sess); saveErr != nil {
				logger.Error("save restored session", "err", saveErr)
			}
			logger.Info("session restored", "email", sess.Email)
		}
	}

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

	initialState := tui.AppState{
		Config: tui.Config{
			Theme:       appCfg.Theme,
			ServerURL:   appCfg.ServerURL,
			OfflineMode: appCfg.OfflineMode,
		},
		Session: sessionState,
	}

	logger.Info("MovieTracker CLI ready", "phase", 8, "database", dbPath)

	program := tea.NewProgram(tui.New(tui.Options{
		MovieService: svc,
		Auth:         &authAdapter{authClient},
		State:        initialState,
		SaveConfig: func(cfg tui.Config) error {
			return config.SaveConfig(config.Config{
				Theme:       cfg.Theme,
				ServerURL:   cfg.ServerURL,
				OfflineMode: cfg.OfflineMode,
			})
		},
		SaveSession: func(state tui.SessionState) error {
			return config.SaveSession(config.Session{
				AccessToken:  state.AccessToken,
				RefreshToken: state.RefreshToken,
				ServerUserID: state.ServerUserID,
				Email:        state.Email,
			})
		},
		ClearSession: config.ClearSession,
	}), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		logger.Error("tui failed", "err", err)
		os.Exit(1)
	}
}

func tuiSessionFromConfig(sess config.Session) tui.SessionState {
	return tui.SessionState{
		AccessToken:   sess.AccessToken,
		RefreshToken:  sess.RefreshToken,
		ServerUserID:  sess.ServerUserID,
		Email:         sess.Email,
		Authenticated: sess.RefreshToken != "" && sess.Email != "",
	}
}
