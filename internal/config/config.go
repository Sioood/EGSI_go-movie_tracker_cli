package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appName         = "movietracker"
	configFileName  = "config.json"
	stateFileName   = "state.json"
	sessionFileName = "session.json"
	dirPerm         = 0o700
	filePerm        = 0o600
)

// Config holds non-sensitive user preferences.
type Config struct {
	Theme       string `json:"theme"`
	ServerURL   string `json:"server_url"`
	OfflineMode bool   `json:"offline_mode"`
}

// State holds persisted application UI state (no movie data).
type State struct {
	LastRoute  string `json:"last_route,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Sort       string `json:"sort,omitempty"`
	LastSyncAt string `json:"last_sync_at,omitempty"`
}

// Session holds auth tokens and server user identity.
type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ServerUserID string `json:"server_user_id"`
	Email        string `json:"email"`
}

// DefaultConfig returns the initial preferences when no config file exists.
func DefaultConfig() Config {
	return Config{
		Theme:       "midnight",
		ServerURL:   "http://localhost:8080",
		OfflineMode: true,
	}
}

// DefaultState returns empty persisted UI state.
func DefaultState() State {
	return State{
		LastRoute: string(RouteSplash),
		Filter:    "all",
		Sort:      "title",
	}
}

// Route names used in persisted state.
type Route string

const (
	RouteSplash      Route = "splash"
	RouteMainMenu    Route = "main_menu"
	RouteMovieList   Route = "movie_list"
	RouteMovieForm   Route = "movie_form"
	RouteMovieDetail Route = "movie_detail"
	RouteStats       Route = "stats"
	RouteSettings    Route = "settings"
	RouteLogin       Route = "login"
	RouteRegister    Route = "register"
	RouteHelp        Route = "help"
)

// Dir returns the XDG config directory path (~/.config/movietracker), creating it with 0700 if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	path := filepath.Join(base, appName)
	if err := os.MkdirAll(path, dirPerm); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := migrateLegacyYAML(path); err != nil {
		return "", err
	}
	return path, nil
}

// LoadConfig reads config.json or returns defaults when the file is absent.
func LoadConfig() (Config, error) {
	dir, err := Dir()
	if err != nil {
		return Config{}, err
	}
	return loadConfigFrom(filepath.Join(dir, configFileName))
}

func loadConfigFrom(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	normalizeConfig(&cfg)
	return cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Theme == "" {
		cfg.Theme = DefaultConfig().Theme
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultConfig().ServerURL
	}
}

// SaveConfig writes config.json with 0600 permissions.
func SaveConfig(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	normalizeConfig(&cfg)
	return writeJSON(filepath.Join(dir, configFileName), cfg)
}

// LoadState reads state.json or returns defaults when the file is absent.
func LoadState() (State, error) {
	dir, err := Dir()
	if err != nil {
		return State{}, err
	}
	return loadStateFrom(filepath.Join(dir, stateFileName))
}

func loadStateFrom(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultState(), nil
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}
	normalizeState(&state)
	return state, nil
}

func normalizeState(state *State) {
	if state.LastRoute == "" {
		state.LastRoute = string(RouteSplash)
	}
	if state.Filter == "" {
		state.Filter = "all"
	}
	if state.Sort == "" {
		state.Sort = "title"
	}
}

// SaveState writes state.json with 0600 permissions.
func SaveState(state State) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	normalizeState(&state)
	return writeJSON(filepath.Join(dir, stateFileName), state)
}

// LoadSession reads session.json or returns an empty session when absent.
func LoadSession() (Session, error) {
	dir, err := Dir()
	if err != nil {
		return Session{}, err
	}
	return loadSessionFrom(filepath.Join(dir, sessionFileName))
}

func loadSessionFrom(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, nil
		}
		return Session{}, fmt.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return Session{}, fmt.Errorf("parse session: %w", err)
	}
	return sess, nil
}

// SaveSession writes session.json with 0600 permissions.
// An empty session removes the file.
func SaveSession(sess Session) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, sessionFileName)
	if sess.AccessToken == "" && sess.RefreshToken == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove session: %w", err)
		}
		return nil
	}
	return writeJSON(path, sess)
}

// ClearSession removes the persisted session file.
func ClearSession() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, sessionFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}

// ExportLocal writes config and state JSON files to the config directory.
func ExportLocal(cfg Config, state State) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := SaveConfig(cfg); err != nil {
		return "", err
	}
	if err := SaveState(state); err != nil {
		return "", err
	}
	return dir, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

// LoadConfigFromDir reads config.json from a custom directory (for tests).
func LoadConfigFromDir(dir string) (Config, error) {
	return loadConfigFrom(filepath.Join(dir, configFileName))
}

// SaveConfigToDir writes config.json to a custom directory (for tests).
func SaveConfigToDir(dir string, cfg Config) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	normalizeConfig(&cfg)
	return writeJSON(filepath.Join(dir, configFileName), cfg)
}

// LoadStateFromDir reads state.json from a custom directory (for tests).
func LoadStateFromDir(dir string) (State, error) {
	return loadStateFrom(filepath.Join(dir, stateFileName))
}

// SaveStateToDir writes state.json to a custom directory (for tests).
func SaveStateToDir(dir string, state State) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	normalizeState(&state)
	return writeJSON(filepath.Join(dir, stateFileName), state)
}

// LoadSessionFromDir reads session.json from a custom directory (for tests).
func LoadSessionFromDir(dir string) (Session, error) {
	return loadSessionFrom(filepath.Join(dir, sessionFileName))
}

// SaveSessionToDir writes session.json to a custom directory (for tests).
func SaveSessionToDir(dir string, sess Session) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	path := filepath.Join(dir, sessionFileName)
	if sess.AccessToken == "" && sess.RefreshToken == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove session: %w", err)
		}
		return nil
	}
	return writeJSON(path, sess)
}

// ClearSessionInDir removes session.json from a custom directory (for tests).
func ClearSessionInDir(dir string) error {
	path := filepath.Join(dir, sessionFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}
