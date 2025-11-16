package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type PRRepo struct {
	Logger log.Logger
}

func NewPRRepo(logger log.Logger) repository.PRRepository {
	return &PRRepo{
		Logger: logger,
	}
}

func (r *PRRepo) CreatePR(ctx context.Context, db repository.DBExecutor, pr *domain.PullRequest) error {
	const q = `
INSERT INTO prs (pr_id, pr_name, author_id, status, created_at, merged_at)
VALUES ($1, $2, $3, $4, $5, $6);
`

	var mergedAt any
	if pr.MergedAt != nil {
		mergedAt = *pr.MergedAt
	} else {
		mergedAt = nil
	}

	_, err := db.Exec(ctx, q,
		pr.ID,
		pr.Name,
		pr.AuthorID,
		string(pr.Status),
		pr.CreatedAt,
		mergedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.NewDomainError(domain.ErrorCodePRExists, "pr id already exists")
		}
		r.Logger.Error("pr_create_failed", "pr_id", pr.ID, "err", err)
		return fmt.Errorf("create pr %q: %w", pr.ID, err)
	}

	return nil
}

func (r *PRRepo) GetPRByID(ctx context.Context, db repository.DBExecutor, prID string) (*domain.PullRequest, []string, error) {
	const qPR = `
SELECT pr_id, pr_name, author_id, status, created_at, merged_at
FROM prs
WHERE pr_id = $1;
`

	var (
		id        string
		name      string
		authorID  string
		statusStr string
		createdAt time.Time
		mergedAt  *time.Time
	)

	err := db.QueryRow(ctx, qPR, prID).Scan(&id, &name, &authorID, &statusStr, &createdAt, &mergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.NewDomainError(domain.ErrorCodeNotFound, "pr not found")
		}
		r.Logger.Error("pr_get_failed", "pr_id", prID, "err", err)
		return nil, nil, fmt.Errorf("get pr %q: %w", prID, err)
	}

	pr := &domain.PullRequest{
		ID:                id,
		Name:              name,
		AuthorID:          authorID,
		Status:            domain.PRStatus(statusStr),
		AssignedReviewers: nil, // заполним ниже
		CreatedAt:         createdAt,
		MergedAt:          mergedAt,
	}

	reviewers, err := r.getReviewers(ctx, db, prID)
	if err != nil {
		return nil, nil, err
	}
	pr.AssignedReviewers = reviewers

	return pr, reviewers, nil
}

func (r *PRRepo) GetPRForUpdate(ctx context.Context, db repository.DBExecutor, prID string) (*domain.PullRequest, []string, error) {
	const qPR = `
SELECT pr_id, pr_name, author_id, status, created_at, merged_at
FROM prs
WHERE pr_id = $1
FOR UPDATE;
`

	var (
		id        string
		name      string
		authorID  string
		statusStr string
		createdAt time.Time
		mergedAt  *time.Time
	)

	err := db.QueryRow(ctx, qPR, prID).Scan(&id, &name, &authorID, &statusStr, &createdAt, &mergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.NewDomainError(domain.ErrorCodeNotFound, "pr not found")
		}
		r.Logger.Error("pr_get_for_update_failed", "pr_id", prID, "err", err)
		return nil, nil, fmt.Errorf("get pr for update %q: %w", prID, err)
	}

	pr := &domain.PullRequest{
		ID:                id,
		Name:              name,
		AuthorID:          authorID,
		Status:            domain.PRStatus(statusStr),
		AssignedReviewers: nil,
		CreatedAt:         createdAt,
		MergedAt:          mergedAt,
	}

	reviewers, err := r.getReviewers(ctx, db, prID)
	if err != nil {
		return nil, nil, err
	}
	pr.AssignedReviewers = reviewers

	return pr, reviewers, nil
}

func (r *PRRepo) SetMerged(ctx context.Context, db repository.DBExecutor, pr *domain.PullRequest) error {
	const q = `
UPDATE prs
SET status = $1, merged_at = $2
WHERE pr_id = $3;
`

	var mergedAt any
	if pr.MergedAt != nil {
		mergedAt = *pr.MergedAt
	} else {
		mergedAt = nil
	}

	_, err := db.Exec(ctx, q, string(pr.Status), mergedAt, pr.ID)
	if err != nil {
		r.Logger.Error("pr_set_merged_failed", "pr_id", pr.ID, "err", err)
		return fmt.Errorf("set pr merged %q: %w", pr.ID, err)
	}

	return nil
}

func (r *PRRepo) AssignReviewers(ctx context.Context, db repository.DBExecutor, prID string, reviewerIDs []string) error {
	if len(reviewerIDs) == 0 {
		return nil
	}

	const q = `
INSERT INTO pr_reviewers (pr_id, user_id, slot, assigned_at)
VALUES ($1, $2, $3, now());
`

	for i, userID := range reviewerIDs {
		slot := i + 1 
		_, err := db.Exec(ctx, q, prID, userID, slot)
		if err != nil {
			r.Logger.Error("pr_assign_reviewer_failed", "pr_id", prID, "user_id", userID, "slot", slot, "err", err)
			return fmt.Errorf("assign reviewers for pr %q: %w", prID, err)
		}
	}

	return nil
}

func (r *PRRepo) ReplaceReviewer(ctx context.Context, db repository.DBExecutor, prID string, oldID, newID string) error {
	const q = `
UPDATE pr_reviewers
SET user_id = $1, assigned_at = now()
WHERE pr_id = $2
  AND user_id = $3;
`

	tag, err := db.Exec(ctx, q, newID, prID, oldID)
	if err != nil {
		r.Logger.Error("pr_replace_reviewer_failed", "pr_id", prID, "old_id", oldID, "new_id", newID, "err", err)
		return fmt.Errorf("replace reviewer for pr %q: %w", prID, err)
	}

	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrorCodeNotAssigned, "reviewer is not assigned to this PR")
	}

	return nil
}

func (r *PRRepo) ListPRsByReviewer(ctx context.Context, db repository.DBExecutor, userID string) ([]domain.PullRequest, error) {
	const q = `
SELECT p.pr_id, p.pr_name, p.author_id, p.status
FROM prs p
JOIN pr_reviewers r ON r.pr_id = p.pr_id
WHERE r.user_id = $1;
`

	rows, err := db.Query(ctx, q, userID)
	if err != nil {
		r.Logger.Error("pr_list_by_reviewer_failed", "user_id", userID, "err", err)
		return nil, fmt.Errorf("list prs by reviewer %q: %w", userID, err)
	}
	defer rows.Close()

	var res []domain.PullRequest

	for rows.Next() {
		var (
			id        string
			name      string
			authorID  string
			statusStr string
		)

		if err := rows.Scan(&id, &name, &authorID, &statusStr); err != nil {
			r.Logger.Error("pr_list_by_reviewer_scan_failed", "user_id", userID, "err", err)
			return nil, fmt.Errorf("scan prs by reviewer %q: %w", userID, err)
		}

		pr := domain.PullRequest{
			ID:       id,
			Name:     name,
			AuthorID: authorID,
			Status:   domain.PRStatus(statusStr),
		}
		res = append(res, pr)
	}

	if err := rows.Err(); err != nil {
		r.Logger.Error("pr_list_by_reviewer_rows_err", "user_id", userID, "err", err)
		return nil, fmt.Errorf("iterate prs by reviewer %q: %w", userID, err)
	}

	return res, nil
}

func (r *PRRepo) AddEvent(ctx context.Context, db repository.DBExecutor, event repository.PREvent) error {
	const q = `
INSERT INTO pr_events (pr_id, event_type, actor_user_id, old_user_id, new_user_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6);
`

	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := db.Exec(ctx, q,
		event.PRID,
		string(event.EventType),
		event.ActorUserID,
		event.OldUserID,
		event.NewUserID,
		createdAt,
	)
	if err != nil {
		r.Logger.Error("pr_add_event_failed", "pr_id", event.PRID, "type", event.EventType, "err", err)
		return fmt.Errorf("add event for pr %q: %w", event.PRID, err)
	}

	return nil
}

func (r *PRRepo) getReviewers(ctx context.Context, db repository.DBExecutor, prID string) ([]string, error) {
	const q = `
SELECT user_id
FROM pr_reviewers
WHERE pr_id = $1
ORDER BY slot;
`

	rows, err := db.Query(ctx, q, prID)
	if err != nil {
		r.Logger.Error("pr_get_reviewers_failed", "pr_id", prID, "err", err)
		return nil, fmt.Errorf("get reviewers for pr %q: %w", prID, err)
	}
	defer rows.Close()

	reviewers := make([]string, 0, 2)

	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			r.Logger.Error("pr_get_reviewers_scan_failed", "pr_id", prID, "err", err)
			return nil, fmt.Errorf("scan reviewers for pr %q: %w", prID, err)
		}
		reviewers = append(reviewers, uid)
	}

	if err := rows.Err(); err != nil {
		r.Logger.Error("pr_get_reviewers_rows_err", "pr_id", prID, "err", err)
		return nil, fmt.Errorf("iterate reviewers for pr %q: %w", prID, err)
	}

	return reviewers, nil
}

func (r *PRRepo) GetAssignStats(
	ctx context.Context,
	db repository.DBExecutor,
) (map[string]int, error) {

	q := `
SELECT user_id, COUNT(*) AS cnt
FROM pr_reviewers
GROUP BY user_id;
`

	rows, err := db.Query(ctx, q)
	if err != nil {
		r.Logger.Error("pr_get_stats_failed", "err", err)
		return nil, fmt.Errorf("get assign stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)

	for rows.Next() {
		var uid string
		var count int

		if err := rows.Scan(&uid, &count); err != nil {
			r.Logger.Error("pr_scan_stats_failed", "err", err)
			return nil, fmt.Errorf("scan assign stats: %w", err)
		}

		stats[uid] = count
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows error: %w", rows.Err())
	}

	return stats, nil
}
