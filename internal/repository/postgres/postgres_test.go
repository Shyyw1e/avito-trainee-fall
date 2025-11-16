package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
)

var (
	testPool   *pgxpool.Pool
	testLogger log.Logger
)

func TestMain(m *testing.M) {
	_ = godotenv.Load("../../../.env")
	testLogger = log.New("DEBUG", "repo_test")
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		panic("POSTGRES_DSN is not set for tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	testPool = pool

	code := m.Run()

	testPool.Close()
	os.Exit(code)
}

func truncateAll(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := testPool.Exec(ctx, `
TRUNCATE TABLE pr_events RESTART IDENTITY CASCADE;
TRUNCATE TABLE pr_reviewers RESTART IDENTITY CASCADE;
TRUNCATE TABLE prs RESTART IDENTITY CASCADE;
TRUNCATE TABLE users RESTART IDENTITY CASCADE;
TRUNCATE TABLE teams RESTART IDENTITY CASCADE;
`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}
