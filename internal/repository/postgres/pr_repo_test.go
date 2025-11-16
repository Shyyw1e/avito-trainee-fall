package postgres

import (
	"context"
	"testing"

	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

func newPRRepo() repository.PRRepository {
	return NewPRRepo(testLogger)
}

func TestPRRepo_GetAssignStats(t *testing.T) {
	truncateAll(t)

	ctx := context.Background()
	repo := newPRRepo()


	_, err := testPool.Exec(ctx, `
INSERT INTO teams (team_name, created_at)
VALUES ('backend', now());
`)
	if err != nil {
		t.Fatalf("insert team failed: %v", err)
	}

	_, err = testPool.Exec(ctx, `
INSERT INTO users (user_id, username, team_name, is_active, created_at)
VALUES 
	('u1', 'Alice',  'backend', TRUE,  now()),
	('u2', 'Bob',    'backend', TRUE,  now()),
	('u3', 'Carol',  'backend', TRUE,  now()),
	('u4', 'Dave',   'backend', FALSE, now());
`)
	if err != nil {
		t.Fatalf("insert users failed: %v", err)
	}

	// 3. PR'ы
	_, err = testPool.Exec(ctx, `
INSERT INTO prs (pr_id, pr_name, author_id, status, created_at)
VALUES 
	('pr-1', 'First PR',  'u1', 'OPEN',  now()),
	('pr-2', 'Second PR', 'u1', 'MERGED', now());
`)
	if err != nil {
		t.Fatalf("insert prs failed: %v", err)
	}

	_, err = testPool.Exec(ctx, `
INSERT INTO pr_reviewers (pr_id, user_id, slot, assigned_at)
VALUES
	('pr-1', 'u2', 0, now()),
	('pr-1', 'u3', 1, now()),
	('pr-2', 'u2', 0, now());
`)
	if err != nil {
		t.Fatalf("insert pr_reviewers failed: %v", err)
	}

	stats, err := repo.GetAssignStats(ctx, testPool)
	if err != nil {
		t.Fatalf("GetAssignStats() error = %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("len(stats) = %d, want 2", len(stats))
	}

	if stats["u2"] != 2 {
		t.Errorf("stats[u2] = %d, want 2", stats["u2"])
	}
	if stats["u3"] != 1 {
		t.Errorf("stats[u3] = %d, want 1", stats["u3"])
	}

	if _, ok := stats["u1"]; ok {
		t.Errorf("author u1 should not be in stats, but present with %d", stats["u1"])
	}
	if _, ok := stats["u4"]; ok {
		t.Errorf("inactive user u4 should not be in stats, but present with %d", stats["u4"])
	}
}
