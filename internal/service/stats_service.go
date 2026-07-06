package service

import (
	"context"

	"github.com/movietracker/movie-tracker/internal/domain"
)

type StatsStore interface {
	GetStats(ctx context.Context, userID string) (domain.Stats, error)
}

type StatsService struct {
	store StatsStore
}

func NewStatsService(store StatsStore) *StatsService {
	return &StatsService{store: store}
}

func (s *StatsService) GetStats(ctx context.Context, userID string) (domain.Stats, error) {
	return s.store.GetStats(ctx, userID)
}
