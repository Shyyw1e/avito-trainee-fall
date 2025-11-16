package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type TeamRepo struct {
	Logger log.Logger
}

func NewTeamRepo(logger log.Logger) repository.TeamRepository {
	return &TeamRepo{
		Logger: logger,
	}
}

func (r *TeamRepo) CreateTeam(ctx context.Context, db repository.DBExecutor, team *domain.Team) error {
	const q = `INSERT INTO teams (team_name, created_at) VALUES ($1, now());`

	_, err := db.Exec(ctx, q, team.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.NewDomainError(domain.ErrorCodeTeamExists, "team already exists")
		}

		r.Logger.Error("team_create_failed", "team", team.Name, "err", err)
		return fmt.Errorf("create team %q: %w", team.Name, err)
	}

	return nil
}

func (r *TeamRepo) UpsertUsersForTeam(ctx context.Context, db repository.DBExecutor, members []domain.User) error {
	if len(members) == 0 {
		return nil
	}

	const q = `
INSERT INTO users (user_id, username, team_name, is_active, created_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (user_id) DO UPDATE SET
    username  = EXCLUDED.username,
    team_name = EXCLUDED.team_name,
    is_active = EXCLUDED.is_active;
`
	for _, u := range members {
		_, err := db.Exec(ctx, q, u.ID, u.Name, u.TeamName, u.IsActive)
		if err != nil {
			r.Logger.Error("team_upsert_members_failed", "team", u.TeamName, "user_id", u.ID, "err", err)
			return fmt.Errorf("upsert users for team %q: %w", u.TeamName, err)
		}
	}

	return nil
}

func (r *TeamRepo) GetTeamWithMembers(ctx context.Context, db repository.DBExecutor, teamName string) (*domain.Team, error) {
	const qTeam = `SELECT team_name FROM teams WHERE team_name = $1;`

	var foundTeam string
	if err := db.QueryRow(ctx, qTeam, teamName).Scan(&foundTeam); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewDomainError(domain.ErrorCodeNotFound, "team not found")
		}
		r.Logger.Error("team_get_failed", "team", teamName, "err", err)
		return nil, fmt.Errorf("get team %q: %w", teamName, err)
	}

	const qMembers = `
SELECT user_id, username, team_name, is_active
FROM users
WHERE team_name = $1
ORDER BY user_id;`

	rows, err := db.Query(ctx, qMembers, teamName)
	if err != nil {
		r.Logger.Error("team_get_members_failed", "team", teamName, "err", err)
		return nil, fmt.Errorf("get team members for %q: %w", teamName, err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)

	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.TeamName, &u.IsActive); err != nil {
			r.Logger.Error("team_get_members_scan_failed", "team", teamName, "err", err)
			return nil, fmt.Errorf("scan team members for %q: %w", teamName, err)
		}
	}

	if err := rows.Err(); err != nil {
		r.Logger.Error("team_get_members_rows_err", "team", teamName, "err", err)
		return nil, fmt.Errorf("iterate team members for %q: %w", teamName, err)
	}

	newTeam, err := domain.NewTeam(teamName, users)
	if err != nil {
		r.Logger.Error("team_domain_build_failed", "team", teamName, "err", err)
		return nil, fmt.Errorf("build domain team for %q: %w", teamName, err)
	}

	return newTeam, nil
}

func (r *TeamRepo) DeactivateUsersByTeam(ctx context.Context, db repository.DBExecutor, teamName string) (int, error) {
	const q = `
UPDATE users
SET is_active = FALSE
WHERE team_name = $1
  AND is_active = TRUE;
`
	tag, err := db.Exec(ctx, q, teamName)
	if err != nil {
		r.Logger.Error("team_deactivate_users_failed", "team", teamName, "err", err)
		return 0, fmt.Errorf("deactivate users by team %q: %w", teamName, err)
	}

	affected := int(tag.RowsAffected())
	return affected, nil
}
