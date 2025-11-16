package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shyyw1e/avito-trainee-fall/internal/http"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/config"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/db"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository/postgres"
	"github.com/Shyyw1e/avito-trainee-fall/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config error: %v\n", err)
		os.Exit(1)
	}

	logger := log.New(cfg.LogLevel, "pull_requester_main")
	logger.Info("app_start", "http_port", cfg.HTTPPort)

	pool, err := openDB(ctx, cfg, logger)
	if err != nil {
		logger.Error("db_connect_failed", "err", err)
		os.Exit(1)
	}
	defer db.Close(pool, logger)

	txManager := usecase.NewPgxTxManager(pool, logger)

	teamRepo := postgres.NewTeamRepo(logger)
	userRepo := postgres.NewUserRepo(logger)
	prRepo := postgres.NewPRRepo(logger)

	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	teamSvc := usecase.NewTeamService(teamRepo, userRepo, txManager, logger)
	userSvc := usecase.NewUserService(userRepo, prRepo, logger)
	prSvc := usecase.NewPRService(prRepo, userRepo, teamRepo, txManager, randSrc, logger)
	statsSvc := usecase.NewStatsService(prRepo, logger)

	apiServer := httpapi.NewServer(
		teamSvc,
		userSvc,
		prSvc,
		statsSvc,
		pool, 
		logger,
	)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Use(requestLoggerMiddleware(logger))
	r.Mount("/", apiServer.Handler())

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("http_server_listen", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown_signal_received")
	case err := <-errCh:
		logger.Error("http_server_failed", "err", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("http_server_shutting_down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http_server_shutdown_error", "err", err)
	} else {
		logger.Info("http_server_stopped")
	}
}

func openDB(ctx context.Context, cfg *config.Config, logger log.Logger) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	return pool, nil
}


func requestLoggerMiddleware(logger log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			duration := time.Since(start)
			logger.Info(
				"http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}
