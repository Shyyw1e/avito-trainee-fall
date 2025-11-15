package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/config"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, cfg *config.Config, logger log.Logger) (*pgxpool.Pool, error) {
	dsn := cfg.DBase.DSN
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("empty dsn")
	}

	clearedDSN := clearSecret(dsn)

	pgcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Error("db_parse_config_failed", "dsn", clearedDSN, "err", err)
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	pgcfg.MaxConns = int32(cfg.DBase.MaxOpenConns)
	pgcfg.MinConns = int32(cfg.DBase.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(ctx, pgcfg)
	if err != nil {
		logger.Error("db_open_failed", "dsn", clearedDSN, "err", err)
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		logger.Error("db_ping_failed", "dsn", clearedDSN, "err", err)
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	logger.Info("db_connected", "dsn", clearedDSN)
	return pool, nil
}

func Close(pool *pgxpool.Pool, logger log.Logger) {
	if pool == nil {
		return
	}
	pool.Close()
	logger.Info("db_closed")
}

func clearSecret(s string) string {
	parts := strings.Split(s, ":")
	if len(parts) < 3 {
		return s
	}
	passPart := strings.Split(parts[2], "@")
	if len(passPart) < 2 {
		return s
	}
	password := passPart[0]
	if password == "" {
		return s
	}
	return strings.Replace(s, password, "****", 1)
}
