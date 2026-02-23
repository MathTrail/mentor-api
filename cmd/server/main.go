// @title       Mentor API
// @version     1.0
// @BasePath    /

package main

import (
	"context"

	"github.com/MathTrail/mentor-api/internal/app"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/logging"
	"github.com/MathTrail/mentor-api/internal/observability"
	"github.com/MathTrail/mentor-api/internal/version"
	"go.uber.org/zap"

	_ "github.com/MathTrail/mentor-api/docs"
)

func main() {
	// 1. Single point of config and logger creation.
	cfg := config.Load()
	logger := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 2. Observability stack (tracing, metrics, profiling).
	obs := observability.New(cfg, logger)
	if err := obs.Init(); err != nil {
		logger.Fatal("failed to initialize observability", zap.Error(err))
	}
	// Shutdown context is created at exit time so the deadline starts
	// only when the process is actually terminating.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		obs.Shutdown(ctx)
	}()

	// 3. DI container (Dapr client, DB, repositories, router).
	container, err := app.NewContainer(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}

	logger.Info("starting mentor-api server",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("date", version.Date),
		zap.String("port", cfg.ServerPort),
	)

	// 4. HTTP server with graceful shutdown.
	srv := app.NewServer(container)
	if err := srv.Run(); err != nil {
		logger.Fatal("server runtime error", zap.Error(err))
	}
}
