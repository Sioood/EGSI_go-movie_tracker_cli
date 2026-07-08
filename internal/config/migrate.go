package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const legacyDirName = ".movietracker"

func migrateLegacyYAML(targetDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	legacyDir := filepath.Join(home, legacyDirName)
	if _, err := os.Stat(legacyDir); os.IsNotExist(err) {
		return nil
	}

	newConfigPath := filepath.Join(targetDir, configFileName)
	if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
		if cfg, ok, err := readLegacyConfig(legacyDir); err != nil {
			return err
		} else if ok {
			if err := writeJSON(newConfigPath, cfg); err != nil {
				return err
			}
		}
	}

	newSessionPath := filepath.Join(targetDir, sessionFileName)
	if _, err := os.Stat(newSessionPath); os.IsNotExist(err) {
		if sess, ok, err := readLegacySession(legacyDir); err != nil {
			return err
		} else if ok {
			if err := writeJSON(newSessionPath, sess); err != nil {
				return err
			}
		}
	}

	archivePath := legacyDir + ".migrated"
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		_ = os.Rename(legacyDir, archivePath)
	}
	return nil
}

func readLegacyConfig(legacyDir string) (Config, bool, error) {
	path := filepath.Join(legacyDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("read legacy config: %w", err)
	}
	var legacy struct {
		Theme       string `yaml:"theme"`
		ServerURL   string `yaml:"server_url"`
		OfflineMode bool   `yaml:"offline_mode"`
	}
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return Config{}, false, fmt.Errorf("parse legacy config: %w", err)
	}
	cfg := Config{
		Theme:       legacy.Theme,
		ServerURL:   legacy.ServerURL,
		OfflineMode: legacy.OfflineMode,
	}
	normalizeConfig(&cfg)
	return cfg, true, nil
}

func readLegacySession(legacyDir string) (Session, bool, error) {
	path := filepath.Join(legacyDir, "session.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Session{}, false, nil
		}
		return Session{}, false, fmt.Errorf("read legacy session: %w", err)
	}
	var legacy Session
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return Session{}, false, fmt.Errorf("parse legacy session: %w", err)
	}
	return legacy, true, nil
}
