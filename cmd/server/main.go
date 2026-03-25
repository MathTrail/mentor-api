// @title       Mentor API
// @version     1.0
// @BasePath    /

package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/MathTrail/mentor-api/internal/app"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/logger"
	"github.com/MathTrail/mentor-api/internal/observability"
	"github.com/MathTrail/mentor-api/internal/version"
	"go.uber.org/zap"

	_ "github.com/MathTrail/mentor-api/docs" // registers Swagger spec via swag-generated init()
)

func main() {
	// 1. Single point of config and logger creation.
	cfg := config.Load()
	logger := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 2. Root context: cancelled on SIGINT or SIGTERM.
	// Created first so it can be passed into every subsystem.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 3. Observability stack (tracing, metrics, profiling).
	obs := observability.New(cfg, logger)
	if err := obs.Init(ctx); err != nil {
		logger.Fatal("failed to initialize observability", zap.Error(err))
	}
	// Shutdown context is created at exit time so the deadline starts
	// only when the process is actually terminating.
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout) // NOSONAR: intentional fresh context — parent ctx is already cancelled at this point
		defer cancel()
		obs.Shutdown(shutCtx)
	}()

	// 4. DI container (DB, repositories, router).
	container, err := app.NewContainer(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}
	defer container.Close()

	logger.Info("starting mentor-api server",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("date", version.Date),
		zap.String("port", cfg.ServerPort),
	)

	// 5. Run background workers and HTTP server under a shared errgroup so that:
	//    - container.Close() (deferred above) only runs after both have exited;
	//    - if any component fails, gCtx is cancelled and the others begin graceful shutdown.
	g, gCtx := errgroup.WithContext(ctx)

	for _, w := range container.Workers {
		w := w
		g.Go(func() error { return w.Start(gCtx) })
	}

	srv := app.NewServer(container)
	g.Go(func() error {
		return srv.Run(gCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application stopped with error", zap.Error(err))
	}
	logger.Info("mentor-api stopped gracefully")
}
