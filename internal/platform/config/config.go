package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type DB struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

type Config struct {
	HTTPPort          int
	DBase             DB
	MigrationDir      string
	LogLevel          string
	AdminTokens       []string
	UserTokens        []string
	RequestTimeout    time.Duration
	ReadHeaderTimeout time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	httpPortStr := getEnv("HTTP_PORT", "8080")
	httpPort, err := strconv.Atoi(httpPortStr)
	if err != nil || httpPort < 1 || httpPort > 65535 {
		return nil, fmt.Errorf("invalid HTTP_PORT %q: %w", httpPortStr, err)
	}
	cfg.HTTPPort = httpPort

	dsn := strings.TrimSpace(os.Getenv("DB_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	}
	if dsn == "" {
		return nil, errors.New("DB_DSN or POSTGRES_DSN must be set")
	}
	cfg.DBase.DSN = dsn

	maxOpenStr := getEnv("DB_MAX_OPEN_CONNS", "20")
	maxOpen, err := strconv.Atoi(maxOpenStr)
	if err != nil || maxOpen <= 0 {
		maxOpen = 20
	}
	cfg.DBase.MaxOpenConns = maxOpen

	maxIdleStr := getEnv("DB_MAX_IDLE_CONNS", "10")
	maxIdle, err := strconv.Atoi(maxIdleStr)
	if err != nil || maxIdle < 0 {
		maxIdle = 10
	}
	cfg.DBase.MaxIdleConns = maxIdle

	migrationDir := getEnv("MIGRATION_DIR", "./migrations")
	if migrationDir == "" {
		migrationDir = getEnv("MIGRATIONS_DIR", "./migrations")
	}
	if migrationDir == "" {
		return nil, errors.New("MIGRATION_DIR must be set")
	}
	if fi, err := os.Stat(migrationDir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("migration dir %q does not exist or is not a directory", migrationDir)
	}
	absDir, err := filepath.Abs(migrationDir)
	if err == nil {
		migrationDir = absDir
	}
	cfg.MigrationDir = migrationDir

	logLevel := strings.ToLower(strings.TrimSpace(getEnv("LOG_LEVEL", "debug")))
	if logLevel == "" {
		logLevel = "debug"
	}
	cfg.LogLevel = logLevel

	reqTimeoutMS := parseIntWithDefault(getEnv("REQUEST_TIMEOUT_MS", "3000"), 3000)
	readHeaderTimeoutMS := parseIntWithDefault(getEnv("READ_HEADER_TIMEOUT_MS", "1000"), 1000)

	cfg.RequestTimeout = time.Duration(reqTimeoutMS) * time.Millisecond
	cfg.ReadHeaderTimeout = time.Duration(readHeaderTimeoutMS) * time.Millisecond

	cfg.AdminTokens = parseCSV(os.Getenv("ADMIN_TOKENS"))
	cfg.UserTokens = parseCSV(os.Getenv("USER_TOKENS"))

	return cfg, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func parseIntWithDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return def
	}
	return n
}

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
