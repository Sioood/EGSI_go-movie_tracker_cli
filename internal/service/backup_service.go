package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/repository"
)

type BackupService struct {
	repo *repository.BackupRepository
}

func NewBackupService(repo *repository.BackupRepository) *BackupService {
	return &BackupService{repo: repo}
}

type BackupSnapshot struct {
	Config   config.Config `json:"config"`
	State    config.State  `json:"state"`
	SyncedAt time.Time     `json:"synced_at"`
}

func (s *BackupService) ExportConfig(ctx context.Context, userID string) (config.Config, error) {
	backup, err := s.repo.Get(ctx, userID)
	if err != nil {
		return config.Config{}, err
	}
	var cfg config.Config
	if err := json.Unmarshal([]byte(backup.Config), &cfg); err != nil {
		return config.DefaultConfig(), nil
	}
	if backup.Config == "" || backup.Config == "{}" {
		return config.DefaultConfig(), nil
	}
	return cfg.WithoutSecrets(), nil
}

func (s *BackupService) ImportConfig(ctx context.Context, userID string, cfg config.Config) error {
	cfg = cfg.WithoutSecrets()
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return s.repo.UpsertConfig(ctx, userID, string(data))
}

func (s *BackupService) ExportState(ctx context.Context, userID string) (config.State, error) {
	backup, err := s.repo.Get(ctx, userID)
	if err != nil {
		return config.State{}, err
	}
	var state config.State
	if err := json.Unmarshal([]byte(backup.State), &state); err != nil {
		return config.DefaultState(), nil
	}
	if backup.State == "" || backup.State == "{}" {
		return config.DefaultState(), nil
	}
	return state, nil
}

func (s *BackupService) ImportState(ctx context.Context, userID string, state config.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return s.repo.UpsertState(ctx, userID, string(data))
}

func (s *BackupService) ExportSnapshot(ctx context.Context, userID string) (BackupSnapshot, error) {
	backup, err := s.repo.Get(ctx, userID)
	if err != nil {
		return BackupSnapshot{}, err
	}

	snapshot := BackupSnapshot{SyncedAt: backup.UpdatedAt}
	if backup.UpdatedAt.IsZero() {
		snapshot.SyncedAt = time.Now().UTC()
	}

	if backup.Config != "" && backup.Config != "{}" {
		_ = json.Unmarshal([]byte(backup.Config), &snapshot.Config)
	} else {
		snapshot.Config = config.DefaultConfig()
	}
	if backup.State != "" && backup.State != "{}" {
		_ = json.Unmarshal([]byte(backup.State), &snapshot.State)
	} else {
		snapshot.State = config.DefaultState()
	}
	snapshot.Config = snapshot.Config.WithoutSecrets()
	return snapshot, nil
}

func (s *BackupService) ImportSnapshot(ctx context.Context, userID string, snapshot BackupSnapshot) error {
	snapshot.Config = snapshot.Config.WithoutSecrets()
	configJSON, err := json.Marshal(snapshot.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	stateJSON, err := json.Marshal(snapshot.State)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return s.repo.UpsertFull(ctx, userID, string(configJSON), string(stateJSON))
}
