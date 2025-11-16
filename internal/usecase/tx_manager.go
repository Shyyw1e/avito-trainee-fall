package usecase

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/db"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, exec repository.DBExecutor) error) error
}

type PgxTxManager struct {
	Pool   *pgxpool.Pool
	Logger log.Logger
}

func NewPgxTxManager(pool *pgxpool.Pool, logger log.Logger) *PgxTxManager {
	return &PgxTxManager{
		Pool:   pool,
		Logger: logger,
	}
}

func (m *PgxTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, exec repository.DBExecutor) error) error {
	return db.WithTx(ctx, m.Pool, m.Logger, func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, tx)
	})
}
