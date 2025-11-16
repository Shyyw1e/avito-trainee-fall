package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type UserRepo struct {
	Logger log.Logger
}

func NewUserRepo(logger log.Logger) repository.UserRepository {
	return &UserRepo{
		Logger: logger,
	}
}

func (r *UserRepo) GetUserByID(ctx context.Context, db repository.DBExecutor, userID string) (*domain.User, error) {
	const q = `
SELECT user_id, username, team_name, is_active
FROM users
WHERE user_id = $1;
`

	var (
		id       string
		username string
		teamName string
		isActive bool
	)

	err := db.QueryRow(ctx, q, userID).Scan(&id, &username, &teamName, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewDomainError(domain.ErrorCodeNotFound, "user not found")
		}
		r.Logger.Error("user_get_failed", "user_id", userID, "err", err)
		return nil, fmt.Errorf("get user %q: %w", userID, err)
	}

	u := &domain.User{
		ID:       id,
		Name:     username,
		TeamName: teamName,
		IsActive: isActive,
	}

	return u, nil
}

func (r *UserRepo) SetUserIsActive(ctx context.Context, db repository.DBExecutor, userID string, isActive bool) (*domain.User, error) {
	const q = `
UPDATE users
SET is_active = $1
WHERE user_id = $2
RETURNING user_id, username, team_name, is_active;
`

	var (
		id       string
		username string
		teamName string
		active   bool
	)

	err := db.QueryRow(ctx, q, isActive, userID).Scan(&id, &username, &teamName, &active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewDomainError(domain.ErrorCodeNotFound, "user not found")
		}
		r.Logger.Error("user_set_is_active_failed", "user_id", userID, "is_active", isActive, "err", err)
		return nil, fmt.Errorf("set user is_active for %q: %w", userID, err)
	}

	u := &domain.User{
		ID:       id,
		Name:     username,
		TeamName: teamName,
		IsActive: active,
	}

	return u, nil
}

func (r *UserRepo) ListUsersByTeam(ctx context.Context, db repository.DBExecutor, teamName string) ([]domain.User, error) {
	const q = `
SELECT user_id, username, team_name, is_active
FROM users
WHERE team_name = $1
ORDER BY user_id;
`

	rows, err := db.Query(ctx, q, teamName)
	if err != nil {
		r.Logger.Error("user_list_by_team_failed", "team", teamName, "err", err)
		return nil, fmt.Errorf("list users by team %q: %w", teamName, err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)

	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.TeamName, &u.IsActive); err != nil {
			r.Logger.Error("user_list_by_team_scan_failed", "team", teamName, "err", err)
			return nil, fmt.Errorf("scan user by team %q: %w", teamName, err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		r.Logger.Error("user_list_by_team_rows_err", "team", teamName, "err", err)
		return nil, fmt.Errorf("iterate users by team %q: %w", teamName, err)
	}

	return users, nil
}
