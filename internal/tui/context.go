package tui

import (
	"context"
	"time"
)

const (
	authRequestTimeout   = 10 * time.Second
	syncRequestTimeout   = 30 * time.Second
	backupRequestTimeout = 15 * time.Second
	tmdbRequestTimeout   = 15 * time.Second
)

func (m Model) appContext() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *Model) shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
