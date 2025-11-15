package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/config"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testDSN(t *testing.T) string {
	t.Helper()

	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}

	t.Skip("no TEST_DB_DSN / DB_DSN / POSTGRES_DSN set, skipping db integration tests")
	return ""
}

func newTestConfig(dsn string) *config.Config {
	return &config.Config{
		DBase: config.DB{
			DSN:          dsn,
			MaxOpenConns: 5,
			MaxIdleConns: 1,
		},
	}
}

func newTestLogger() log.Logger {
	return log.New("debug", "db-test")
}

func TestOpen_Success(t *testing.T) {
	t.Parallel()

	dsn := testDSN(t)
	cfg := newTestConfig(dsn)
	logger := newTestLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, pool)

	defer Close(pool, logger)

	err = pool.Ping(ctx)
	require.NoError(t, err)
}

func ensureTestTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tx_test_items (
			id    SERIAL PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	require.NoError(t, err)
}

func TestWithTx_Commit(t *testing.T) {
	dsn := testDSN(t)
	cfg := newTestConfig(dsn)
	logger := newTestLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer Close(pool, logger)

	ensureTestTable(t, ctx, pool)

	value := "commit-case-" + time.Now().Format(time.RFC3339Nano)

	err = WithTx(ctx, pool, logger, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO tx_test_items (value) VALUES ($1)`, value)
		return err
	})
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM tx_test_items WHERE value = $1`, value).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "row must exist after successful tx commit")
}

func TestWithTx_Rollback(t *testing.T) {
	dsn := testDSN(t)
	cfg := newTestConfig(dsn)
	logger := newTestLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer Close(pool, logger)

	ensureTestTable(t, ctx, pool)

	value := "rollback-case-" + time.Now().Format(time.RFC3339Nano)

	forcedErr := fmt.Errorf("force rollback")

	err = WithTx(ctx, pool, logger, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO tx_test_items (value) VALUES ($1)`, value)
		require.NoError(t, err)

		return forcedErr
	})
	require.Error(t, err)
	require.Equal(t, forcedErr, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM tx_test_items WHERE value = $1`, value).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "row must NOT exist after rollback")
}