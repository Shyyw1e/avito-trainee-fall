package usecase

import (
	"context"
	"fmt"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type TeamService struct {
	Teams  repository.TeamRepository
	Users  repository.UserRepository
	Tx     TxManager
	Logger log.Logger
}

func NewTeamService(
	teams repository.TeamRepository,
	users repository.UserRepository,
	tx TxManager,
	logger log.Logger,
) *TeamService {
	return &TeamService{
		Teams:  teams,
		Users:  users,
		Tx:     tx,
		Logger: logger,
	}
}

func (s *TeamService) AddTeam(
	ctx context.Context,
	teamName string,
	members []domain.User,
) (*domain.Team, error) {
	team, err := domain.NewTeam(teamName, members)
	if err != nil {
		return nil, err
	}

	err = s.Tx.WithTx(ctx, func(ctx context.Context, exec repository.DBExecutor) error {
		if err := s.Teams.CreateTeam(ctx, exec, team); err != nil {
			return err
		}

		if err := s.Teams.UpsertUsersForTeam(ctx, exec, team.Members); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.Logger.Error("team_add_failed", "team", teamName, "err", err)
		return nil, err
	}

	return team, nil
}

func (s *TeamService) GetTeam(
	ctx context.Context,
	teamName string,
	exec repository.DBExecutor,
) (*domain.Team, error) {
	team, err := s.Teams.GetTeamWithMembers(ctx, exec, teamName)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) MassDeactivateTeam(
	ctx context.Context,
	teamName string,
) (int, error) {
	var affected int

	err := s.Tx.WithTx(ctx, func(ctx context.Context, exec repository.DBExecutor) error {
		n, err := s.Teams.DeactivateUsersByTeam(ctx, exec, teamName)
		if err != nil {
			return err
		}
		affected = n
		return nil
	})
	if err != nil {
		s.Logger.Error("team_mass_deactivate_failed", "team", teamName, "err", err)
		return 0, fmt.Errorf("mass deactivate team %q: %w", teamName, err)
	}

	return affected, nil
}
