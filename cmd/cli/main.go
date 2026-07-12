package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/database"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/logging"
	"github.com/movietracker/movie-tracker/internal/repository"
	"github.com/movietracker/movie-tracker/internal/service"
	appsync "github.com/movietracker/movie-tracker/internal/sync"
	"github.com/movietracker/movie-tracker/internal/tmdb"
	"github.com/movietracker/movie-tracker/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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

type backupAdapter struct {
	*client.BackupClient
}

func (b *backupAdapter) ExportSnapshot(ctx context.Context, accessToken string) (tui.BackupSnapshot, error) {
	snapshot, err := b.BackupClient.ExportSnapshot(ctx, accessToken)
	if err != nil {
		return tui.BackupSnapshot{}, err
	}
	return tui.BackupSnapshot{Config: snapshot.Config, State: snapshot.State}, nil
}

func (b *backupAdapter) ImportSnapshot(ctx context.Context, accessToken string, snapshot tui.BackupSnapshot) error {
	return b.BackupClient.ImportSnapshot(ctx, accessToken, service.BackupSnapshot{
		Config:   snapshot.Config,
		State:    snapshot.State,
		SyncedAt: time.Now().UTC(),
	})
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
	return tui.SyncResult{PendingCount: result.PendingCount, ConflictCount: result.ConflictCount}, err
}

func (a *syncRunnerAdapter) PendingCount(ctx context.Context) (int, error) {
	return a.Service.PendingCount(ctx)
}

func (a *syncRunnerAdapter) ConflictCount(ctx context.Context) (int, error) {
	return a.Service.ConflictCount(ctx)
}

func (a *syncRunnerAdapter) ListConflicts(ctx context.Context) ([]domain.SyncConflict, error) {
	return a.Service.ListConflicts(ctx)
}

func (a *syncRunnerAdapter) ResolveConflict(ctx context.Context, id, choice string) error {
	return a.Service.ResolveConflict(ctx, id, choice)
}

func (a *syncRunnerAdapter) GetDeviceName(ctx context.Context, deviceID string) (string, error) {
	return a.Service.GetDeviceName(ctx, deviceID)
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
	appCfg = ensureDeviceConfig(appCfg)
	if err := config.SaveConfig(appCfg); err != nil {
		logger.Warn("save device config", "err", err)
	}

	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if appCfg.TMDBAPIKey != "" {
		tmdbAPIKey = appCfg.TMDBAPIKey
	}

	appState, err := config.LoadState()
	if err != nil {
		logger.Error("load state", "err", err)
		os.Exit(1)
	}

	sess, err := config.LoadSession()
	if err != nil {
		logger.Error("load session", "err", err)
		os.Exit(1)
	}

	authClient := client.NewAuthClient(appCfg.ServerURL)
	backupClient := client.NewBackupClient(appCfg.ServerURL)
	externalClient := client.NewExternalClient(appCfg.ServerURL)
	directTMDB := tmdb.NewClient(tmdbAPIKey)
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
	tmdbCacheRepository := repository.NewTMDBCacheRepository(db)

	movieService := service.NewMovieService(movieRepository, watchEntryRepository)
	statsService := service.NewStatsService(statsRepository)
	exportService := service.NewExportService(movieService)
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
		func() string {
			return appCfg.DeviceID
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
		func() string {
			return appCfg.DeviceID
		},
		func() bool {
			current := runtime.currentSession()
			return !runtime.offline.Load() && current.Authenticated
		},
		bridge.Request,
	)

	tmdbSearch := service.NewTMDBSearchService(
		externalClient,
		directTMDB,
		tmdbCacheRepository,
		func() string {
			return runtime.currentSession().AccessToken
		},
		func() bool {
			current := runtime.currentSession()
			return !runtime.offline.Load() && current.Authenticated
		},
	)

	initialState := tui.AppState{
		Config: tui.Config{
			Theme:       appCfg.Theme,
			ServerURL:   appCfg.ServerURL,
			OfflineMode: appCfg.OfflineMode,
			DeviceID:    appCfg.DeviceID,
			DeviceName:  appCfg.DeviceName,
		},
		Session: sessionState,
	}

	logger.Info("MovieTracker CLI ready", "database", dbPath)

	model := tui.New(tui.Options{
		MovieService:  hybridClient,
		Auth:          &authAdapter{authClient},
		Backup:        &backupAdapter{backupClient},
		SyncRunner:    &syncRunnerAdapter{syncService},
		TMDBSearch:    tmdbSearch,
		InitialRoute:  tui.ParseRoute(appState.LastRoute),
		InitialFilter: domainMovieFilter(appState.Filter),
		InitialSort:   domainMovieSort(appState.Sort),
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
			backupClient.SetBaseURL(cfg.ServerURL)
			externalClient.SetBaseURL(cfg.ServerURL)
			return config.SaveConfig(config.Config{
				Theme:       cfg.Theme,
				ServerURL:   cfg.ServerURL,
				OfflineMode: cfg.OfflineMode,
				DeviceID:    cfg.DeviceID,
				DeviceName:  cfg.DeviceName,
			})
		},
		SaveState: func(state config.State) error {
			return config.SaveState(state)
		},
		ExportLocal: func(snapshot tui.BackupSnapshot) (string, error) {
			return config.ExportLocal(snapshot.Config, snapshot.State)
		},
		ExportMovies: func(format string) (string, error) {
			userID, err := syncService.UserID(context.Background())
			if err != nil {
				userID = appsync.LocalUserID
			}
			switch format {
			case "json":
				return exportService.ExportJSON(context.Background(), userID)
			case "csv":
				return exportService.ExportCSV(context.Background(), userID)
			default:
				return "", fmt.Errorf("format d'export inconnu: %s", format)
			}
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

func domainMovieFilter(value string) domain.MovieFilter {
	switch domain.MovieFilter(value) {
	case domain.MovieFilterWatched, domain.MovieFilterUnwatched, domain.MovieFilterRated, domain.MovieFilterUnrated:
		return domain.MovieFilter(value)
	default:
		return domain.MovieFilterAll
	}
}

func domainMovieSort(value string) domain.MovieSort {
	switch domain.MovieSort(value) {
	case domain.MovieSortDate, domain.MovieSortRating:
		return domain.MovieSort(value)
	default:
		return domain.MovieSortTitle
	}
}

func ensureDeviceConfig(cfg config.Config) config.Config {
	if cfg.DeviceID == "" {
		cfg.DeviceID = uuid.NewString()
	}
	return cfg
}
