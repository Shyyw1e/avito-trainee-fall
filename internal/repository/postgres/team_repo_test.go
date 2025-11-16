package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	//"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

func newTeamRepo() *TeamRepo {
	return &TeamRepo{
		Logger: testLogger,
	}
}

func TestTeamRepo_CreateAndGetTeamWithMembers_Success(t *testing.T) {
	truncateAll(t)

	ctx := context.Background()
	repo := newTeamRepo()

	users := []domain.User{
		{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: false},
	}

	team, err := domain.NewTeam("backend", users)
	if err != nil {
		t.Fatalf("NewTeam() error = %v", err)
	}

	if err := repo.CreateTeam(ctx, testPool, team); err != nil {
		t.Fatalf("CreateTeam() error = %v", err)
	}

	if err := repo.UpsertUsersForTeam(ctx, testPool, users); err != nil {
		t.Fatalf("UpsertUsersForTeam() error = %v", err)
	}
	got, err := repo.GetTeamWithMembers(ctx, testPool, "backend")
	if err != nil {
		t.Fatalf("GetTeamWithMembers() error = %v", err)
	}

	if got.Name != "backend" {
		t.Errorf("team name = %v, want %v", got.Name, "backend")
	}

	if len(got.Members) != 2 {
		t.Fatalf("members len = %d, want 2", len(got.Members))
	}

	if got.Members[0].ID != "u1" || got.Members[1].ID != "u2" {
		t.Errorf("unexpected members: %+v", got.Members)
	}
}

func TestTeamRepo_CreateTeam_AlreadyExists(t *testing.T) {
	truncateAll(t)

	ctx := context.Background()
	repo := newTeamRepo()

	team := &domain.Team{
		Name:    "backend",
		Members: nil,
	}

	if err := repo.CreateTeam(ctx, testPool, team); err != nil {
		t.Fatalf("CreateTeam() first call error = %v", err)
	}
	err := repo.CreateTeam(ctx, testPool, team)
	if err == nil {
		t.Fatalf("expected error on duplicate team, got nil")
	}

	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T: %v", err, err)
	}

	if domainErr.Code != domain.ErrorCodeTeamExists {
		t.Errorf("error code = %v, want %v", domainErr.Code, domain.ErrorCodeTeamExists)
	}
}

func TestTeamRepo_DeactivateUsersByTeam(t *testing.T) {
	truncateAll(t)

	ctx := context.Background()
	repo := newTeamRepo()

	users := []domain.User{
		{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: true},
		{ID: "u3", Name: "Carol", TeamName: "other", IsActive: true},
	}

	teamBackend, _ := domain.NewTeam("backend", users[:2])
	teamOther, _ := domain.NewTeam("other", users[2:])

	if err := repo.CreateTeam(ctx, testPool, teamBackend); err != nil {
		t.Fatalf("CreateTeam(backend) error = %v", err)
	}
	if err := repo.CreateTeam(ctx, testPool, teamOther); err != nil {
		t.Fatalf("CreateTeam(other) error = %v", err)
	}

	if err := repo.UpsertUsersForTeam(ctx, testPool, users); err != nil {
		t.Fatalf("UpsertUsersForTeam() error = %v", err)
	}

	affected, err := repo.DeactivateUsersByTeam(ctx, testPool, "backend")
	if err != nil {
		t.Fatalf("DeactivateUsersByTeam() error = %v", err)
	}

	if affected != 2 {
		t.Errorf("affected = %d, want 2", affected)
	}

	_ = time.Now()
}
