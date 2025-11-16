package usecase

import (
	"context"
	"fmt"

	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type StatsService struct {
	prs    repository.PRRepository
	logger log.Logger
}

func NewStatsService(
	prs repository.PRRepository,
	logger log.Logger,
) *StatsService {
	return &StatsService{
		prs:    prs,
		logger: logger,
	}
}

func (s *StatsService) GetAssignmentsByUser(
	ctx context.Context,
	exec repository.DBExecutor,
) (map[string]int, error) {
	stats, err := s.prs.GetAssignStats(ctx, exec)
	if err != nil {
		s.logger.Error("stats_get_assignments_failed", "err", err)
		return nil, fmt.Errorf("get assignment stats: %w", err)
	}
	return stats, nil
}
