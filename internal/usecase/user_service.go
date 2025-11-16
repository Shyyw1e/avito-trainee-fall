package usecase

import (
	"context"
	"fmt"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type UserService struct {
	Users  repository.UserRepository
	PRs    repository.PRRepository
	Logger log.Logger
}

func NewUserService(
	users repository.UserRepository,
	prs repository.PRRepository,
	logger log.Logger,
) *UserService {
	return &UserService{
		Users:  users,
		PRs:    prs,
		Logger: logger,
	}
}

func (s *UserService) SetUserIsActive(
	ctx context.Context,
	exec repository.DBExecutor,
	userID string,
	isActive bool,
) (*domain.User, error) {
	user, err := s.Users.SetUserIsActive(ctx, exec, userID, isActive)
	if err != nil {
		s.Logger.Error("user_set_is_active_failed", "user_id", userID, "is_active", isActive, "err", err)
		return nil, fmt.Errorf("set user %q is_active=%v: %w", userID, isActive, err)
	}
	return user, nil
}

func (s *UserService) GetUserReviews(
	ctx context.Context,
	exec repository.DBExecutor,
	userID string,
) (string, []domain.PullRequest, error) {
	prs, err := s.PRs.ListPRsByReviewer(ctx, exec, userID)
	if err != nil {
		s.Logger.Error("user_get_reviews_failed", "user_id", userID, "err", err)
		return "", nil, fmt.Errorf("get reviews for user %q: %w", userID, err)
	}

	return userID, prs, nil
}
