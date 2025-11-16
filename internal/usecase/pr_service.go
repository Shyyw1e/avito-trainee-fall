package usecase

import (
	"context"
	"fmt"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type Rand interface {
	Intn(n int) int
}

type PRService struct {
	prs    repository.PRRepository
	users  repository.UserRepository
	teams  repository.TeamRepository
	tx     TxManager
	rand   Rand
	logger log.Logger
}

func NewPRService(
	prs repository.PRRepository,
	users repository.UserRepository,
	teams repository.TeamRepository,
	tx TxManager,
	rand Rand,
	logger log.Logger,
) *PRService {
	return &PRService{
		prs:    prs,
		users:  users,
		teams:  teams,
		tx:     tx,
		rand:   rand,
		logger: logger,
	}
}

func (s *PRService) CreatePRWithAutoAssign(
	ctx context.Context,
	prID, prName, authorID string,
) (*domain.PullRequest, error) {
	var created *domain.PullRequest

	err := s.tx.WithTx(ctx, func(ctx context.Context, exec repository.DBExecutor) error {
		author, err := s.users.GetUserByID(ctx, exec, authorID)
		if err != nil {
			return err
		}

		team, err := s.teams.GetTeamWithMembers(ctx, exec, author.TeamName)
		if err != nil {
			return err
		}

		candidates := make([]domain.User, 0, len(team.Members))
		for _, m := range team.Members {
			if !m.IsActive {
				continue
			}
			if m.ID == author.ID {
				continue
			}
			candidates = append(candidates, m)
		}

		reviewerIDs := chooseReviewers(candidates, s.rand)

		pr, err := domain.NewPullRequest(prID, prName, authorID)
		if err != nil {
			return err
		}

		if err := pr.AssignReviewers(reviewerIDs); err != nil {
			return err
		}

		if err := s.prs.CreatePR(ctx, exec, pr); err != nil {
			return err
		}

		if len(reviewerIDs) > 0 {
			if err := s.prs.AssignReviewers(ctx, exec, pr.ID, reviewerIDs); err != nil {
				return err
			}
		}

		created = pr
		return nil
	})
	if err != nil {
		s.logger.Error("pr_create_usecase_failed", "pr_id", prID, "err", err)
		return nil, err
	}

	if created == nil {
		s.logger.Error("pr_create_usecase_nil_result", "pr_id", prID)
		return nil, fmt.Errorf("internal error: pr %q was not created", prID)
	}

	return created, nil
}


func (s *PRService) MergePR(
	ctx context.Context,
	prID string,
) (*domain.PullRequest, error) {
	var result *domain.PullRequest

	err := s.tx.WithTx(ctx, func(ctx context.Context, exec repository.DBExecutor) error {
		pr, _, err := s.prs.GetPRForUpdate(ctx, exec, prID)
		if err != nil {
			return err
		}

		if pr.Status == domain.PRStatusMerged {
			result = pr
			return nil
		}

		pr.MarkMerged()

		if err := s.prs.SetMerged(ctx, exec, pr); err != nil {
			return err
		}

		result = pr
		return nil
	})
	if err != nil {
		s.logger.Error("pr_merge_usecase_failed", "pr_id", prID, "err", err)
		return nil, err
	}

	return result, nil
}

func (s *PRService) ReassignReviewer(
	ctx context.Context,
	prID, oldReviewerID string,
) (*domain.PullRequest, string, error) {
	var (
		result *domain.PullRequest
		newID  string
	)

	err := s.tx.WithTx(ctx, func(ctx context.Context, exec repository.DBExecutor) error {
		pr, reviewers, err := s.prs.GetPRForUpdate(ctx, exec, prID)
		if err != nil {
			return err
		}

		if !pr.CanModifyReviewers() {
			return domain.NewDomainError(domain.ErrorCodePRMerged, "cannot reassign on merged PR")
		}

		found := false
		for _, rID := range reviewers {
			if rID == oldReviewerID {
				found = true
				break
			}
		}
		if !found {
			return domain.NewDomainError(domain.ErrorCodeNotAssigned, "reviewer is not assigned to this PR")
		}

		oldUser, err := s.users.GetUserByID(ctx, exec, oldReviewerID)
		if err != nil {
			return err
		}

		team, err := s.teams.GetTeamWithMembers(ctx, exec, oldUser.TeamName)
		if err != nil {
			return err
		}

		assignedSet := make(map[string]struct{}, len(reviewers))
		for _, id := range reviewers {
			assignedSet[id] = struct{}{}
		}

		candidates := make([]domain.User, 0, len(team.Members))
		for _, m := range team.Members {
			if !m.IsActive {
				continue
			}
			if m.ID == oldReviewerID {
				continue
			}
			if m.ID == pr.AuthorID {
				continue
			}
			if _, exists := assignedSet[m.ID]; exists {
				continue
			}
			candidates = append(candidates, m)
		}

		if len(candidates) == 0 {
			return domain.NewDomainError(domain.ErrorCodeNoCandidate, "no active replacement candidate in team")
		}

		newID = chooseOne(candidates, s.rand)

		if err := pr.ReplaceReviewer(oldReviewerID, newID); err != nil {
			return err
		}

		if err := s.prs.ReplaceReviewer(ctx, exec, prID, oldReviewerID, newID); err != nil {
			return err
		}

		result = pr
		return nil
	})
	if err != nil {
		s.logger.Error("pr_reassign_usecase_failed", "pr_id", prID, "old_reviewer", oldReviewerID, "err", err)
		return nil, "", err
	}

	return result, newID, nil
}

func chooseReviewers(candidates []domain.User, r Rand) []string {
	n := len(candidates)
	switch n {
	case 0:
		return nil
	case 1:
		return []string{candidates[0].ID}
	default:
		if r == nil {
			return []string{candidates[0].ID, candidates[1].ID}
		}

		i := r.Intn(n)
		j := r.Intn(n - 1)
		if j >= i {
			j++
		}

		return []string{candidates[i].ID, candidates[j].ID}
	}
}

func chooseOne(candidates []domain.User, r Rand) string {
	if len(candidates) == 0 {
		return ""
	}
	if r == nil {
		return candidates[0].ID
	}
	idx := r.Intn(len(candidates))
	return candidates[idx].ID
}
