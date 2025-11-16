package repository

import (
	"context"
	"time"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}


type TeamRepository interface {
	CreateTeam(ctx context.Context, db DBExecutor, team *domain.Team) error
	UpsertUsersForTeam(ctx context.Context, db DBExecutor, members []domain.User) error
	GetTeamWithMembers(ctx context.Context, db DBExecutor, teamName string) (*domain.Team, error)
	DeactivateUsersByTeam(ctx context.Context, db DBExecutor, teamName string) (int, error)
}


type UserRepository interface {
	GetUserByID(ctx context.Context, db DBExecutor, userID string) (*domain.User, error)
	SetUserIsActive(ctx context.Context, db DBExecutor, userID string, isActive bool) (*domain.User, error)
	ListUsersByTeam(ctx context.Context, db DBExecutor, teamName string) ([]domain.User, error)
}


type PRRepository interface {
	CreatePR(ctx context.Context, db DBExecutor, pr *domain.PullRequest) error
	GetAssignStats(ctx context.Context, db DBExecutor) (map[string]int, error)
	GetPRByID(ctx context.Context, db DBExecutor, prID string) (*domain.PullRequest, []string, error)
	GetPRForUpdate(ctx context.Context, db DBExecutor, prID string) (*domain.PullRequest, []string, error)
	SetMerged(ctx context.Context, db DBExecutor, pr *domain.PullRequest) error
	ReplaceReviewer(ctx context.Context, db DBExecutor, prID string, oldID, newID string) error
	AssignReviewers(ctx context.Context, db DBExecutor, prID string, reviewerIDs []string) error
	ListPRsByReviewer(ctx context.Context, db DBExecutor, userID string) ([]domain.PullRequest, error)
	AddEvent(ctx context.Context, db DBExecutor, event PREvent) error
}

type PREventType string

const (
	PREventTypeCreated          PREventType = "CREATED"
	PREventTypeMerged           PREventType = "MERGED"
	PREventTypeReviewerAssigned PREventType = "REVIEWER_ASSIGNED"
	PREventTypeReviewerReplaced PREventType = "REVIEWER_REPLACED"
)

type PREvent struct {
	ID          int64
	PRID        string
	EventType   PREventType
	ActorUserID string
	OldUserID   string
	NewUserID   string
	CreatedAt   time.Time
}
