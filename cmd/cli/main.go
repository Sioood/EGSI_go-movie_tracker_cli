package main

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/logging"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/service"
	appsync "github.com/movietracker/movie-tracker/internal/sync"
	"github.com/movietracker/movie-tracker/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

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

type tokenRefresher struct {
	*client.AuthClient
}

func (t *tokenRefresher) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	pair, err := t.AuthClient.Refresh(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}
	return pair.AccessToken, pair.RefreshToken, nil
}

type syncRunnerAdapter struct {
	*appsync.Service
}

func (a *syncRunnerAdapter) Run(ctx context.Context) (tui.SyncResult, error) {
	result, err := a.Service.Run(ctx)
	return tui.SyncResult{PendingCount: result.PendingCount}, err
}

func (a *syncRunnerAdapter) PendingCount(ctx context.Context) (int, error) {
	return a.Service.PendingCount(ctx)
}

type runtimeState struct {
	offline atomic.Bool
	session atomic.Value // tui.SessionState
}

func (r *runtimeState) currentSession() tui.SessionState {
	if value := r.session.Load(); value != nil {
		return value.(tui.SessionState)
	}
	return tui.SessionState{}
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
	syncRepository := repository.NewSyncRepository(db, movieRepository, watchEntryRepository)

	movieService := service.NewMovieService(movieRepository, watchEntryRepository)
	statsService := service.NewStatsService(statsRepository)
	local := &appsync.LocalService{MovieService: movieService, StatsService: statsService}

	runtime := &runtimeState{}
	runtime.offline.Store(appCfg.OfflineMode)
	runtime.session.Store(sessionState)

	syncClient := client.NewSyncClient(appCfg.ServerURL)
	syncService := appsync.NewService(
		movieService,
		syncRepository,
		syncClient,
		&tokenRefresher{authClient},
		func() appsync.SessionAccess {
			current := runtime.currentSession()
			return appsync.SessionAccess{
				AccessToken:  current.AccessToken,
				RefreshToken: current.RefreshToken,
				ServerUserID: current.ServerUserID,
			}
		},
		func() bool {
			current := runtime.currentSession()
			return !runtime.offline.Load() && current.Authenticated
		},
		func(access, refresh string) {
			current := runtime.currentSession()
			current.AccessToken = access
			current.RefreshToken = refresh
			runtime.session.Store(current)
			_ = config.SaveSession(config.Session{
				AccessToken:  access,
				RefreshToken: refresh,
				ServerUserID: current.ServerUserID,
				Email:        current.Email,
			})
		},
	)

	bridge := &tui.ProgramSyncBridge{}
	hybridClient := appsync.NewHybridClient(
		local,
		local,
		syncService,
		func(ctx context.Context) (string, error) {
			return syncService.UserID(ctx)
		},
		func() bool {
			current := runtime.currentSession()
			return !runtime.offline.Load() && current.Authenticated
		},
		bridge.Request,
	)

	initialState := tui.AppState{
		Config: tui.Config{
			Theme:       appCfg.Theme,
			ServerURL:   appCfg.ServerURL,
			OfflineMode: appCfg.OfflineMode,
		},
		Session: sessionState,
	}

	logger.Info("MovieTracker CLI ready", "phase", 9, "database", dbPath)

	model := tui.New(tui.Options{
		MovieService: hybridClient,
		Auth:         &authAdapter{authClient},
		SyncRunner:   &syncRunnerAdapter{syncService},
		ResolveUserID: func() string {
			userID, err := syncService.UserID(context.Background())
			if err != nil {
				return appsync.LocalUserID
			}
			return userID
		},
		State: initialState,
		SaveConfig: func(cfg tui.Config) error {
			runtime.offline.Store(cfg.OfflineMode)
			authClient.SetBaseURL(cfg.ServerURL)
			syncClient.SetBaseURL(cfg.ServerURL)
			return config.SaveConfig(config.Config{
				Theme:       cfg.Theme,
				ServerURL:   cfg.ServerURL,
				OfflineMode: cfg.OfflineMode,
			})
		},
		SaveSession: func(state tui.SessionState) error {
			runtime.session.Store(state)
			return config.SaveSession(config.Session{
				AccessToken:  state.AccessToken,
				RefreshToken: state.RefreshToken,
				ServerUserID: state.ServerUserID,
				Email:        state.Email,
			})
		},
		ClearSession: func() error {
			runtime.session.Store(tui.SessionState{})
			return config.ClearSession()
		},
	})

	program := tea.NewProgram(model, tea.WithAltScreen())
	bridge.Bind(program)
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
