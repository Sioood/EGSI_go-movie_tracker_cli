package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	dirName        = ".movietracker"
	configFileName = "config.yaml"
	sessionFileName = "session.yaml"
	dirPerm        = 0o700
	filePerm       = 0o600
)

// Config holds non-sensitive user preferences.
type Config struct {
	Theme       string `yaml:"theme"`
	ServerURL   string `yaml:"server_url"`
	OfflineMode bool   `yaml:"offline_mode"`
}

// Session holds auth tokens and server user identity.
type Session struct {
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	ServerUserID string `yaml:"server_user_id"`
	Email        string `yaml:"email"`
}

// DefaultConfig returns the initial preferences when no config file exists.
func DefaultConfig() Config {
	return Config{
		Theme:       "midnight",
		ServerURL:   "http://localhost:8080",
		OfflineMode: true,
	}
}

// Dir returns the config directory path, creating it with 0700 if needed.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	path := filepath.Join(home, dirName)
	if err := os.MkdirAll(path, dirPerm); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return path, nil
}

// LoadConfig reads config.yaml or returns defaults when the file is absent.
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Theme == "" {
		cfg.Theme = DefaultConfig().Theme
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultConfig().ServerURL
	}
	return cfg, nil
}

// SaveConfig writes config.yaml with 0600 permissions.
func SaveConfig(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, configFileName), cfg)
}

// LoadSession reads session.yaml or returns an empty session when absent.
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
	if err := yaml.Unmarshal(data, &sess); err != nil {
		return Session{}, fmt.Errorf("parse session: %w", err)
	}
	return sess, nil
}

// SaveSession writes session.yaml with 0600 permissions.
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
	return writeYAML(path, sess)
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

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

// LoadConfigFromDir reads config.yaml from a custom directory (for tests).
func LoadConfigFromDir(dir string) (Config, error) {
	return loadConfigFrom(filepath.Join(dir, configFileName))
}

// SaveConfigToDir writes config.yaml to a custom directory (for tests).
func SaveConfigToDir(dir string, cfg Config) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return writeYAML(filepath.Join(dir, configFileName), cfg)
}

// LoadSessionFromDir reads session.yaml from a custom directory (for tests).
func LoadSessionFromDir(dir string) (Session, error) {
	return loadSessionFrom(filepath.Join(dir, sessionFileName))
}

// SaveSessionToDir writes session.yaml to a custom directory (for tests).
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
	return writeYAML(path, sess)
}

// ClearSessionInDir removes session.yaml from a custom directory (for tests).
func ClearSessionInDir(dir string) error {
	path := filepath.Join(dir, sessionFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}
