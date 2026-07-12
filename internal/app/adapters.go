package app

import (
	"context"
	"time"

	"github.com/movietracker/movie-tracker/internal/client"
	"github.com/movietracker/movie-tracker/internal/domain"
	"github.com/movietracker/movie-tracker/internal/service"
	appsync "github.com/movietracker/movie-tracker/internal/sync"
	"github.com/movietracker/movie-tracker/internal/tui"
)

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
	auth *client.AuthClient
}

func (t *tokenRefresher) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	pair, err := t.auth.Refresh(ctx, refreshToken)
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
