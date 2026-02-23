// @title       Mentor API
// @version     1.0
// @BasePath    /

package main

import (
	"time"

	"github.com/MathTrail/mentor-api/internal/app"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/logging"
	"github.com/MathTrail/mentor-api/internal/observability"
	"go.uber.org/zap"

	_ "github.com/MathTrail/mentor-api/docs"
)

func main() {
	// 1. Single point of config and logger creation.
	cfg := config.Load()
	logger := logging.NewLogger(cfg.LogLevel)

	// 2. Observability stack (tracing, metrics, profiling).
	obs := observability.New(cfg, logger)
	if err := obs.Init(); err != nil {
		logger.Fatal("failed to initialize observability", zap.Error(err))
	}
	defer obs.Shutdown(5 * time.Second)

	// 3. DI container (Dapr client, DB, repositories, router).
	container, err := app.NewContainer(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}

	logger.Info("starting mentor-api server",
		zap.String("port", cfg.ServerPort),
		zap.String("db_host", cfg.DBHost),
	)

	// 4. HTTP server with graceful shutdown.
	srv := app.NewServer(container)
	if err := srv.Run(); err != nil {
		logger.Fatal("server runtime error", zap.Error(err))
	}
}
