// @title       Mentor API
// @version     1.0
// @BasePath    /

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MathTrail/mentor-api/internal/app"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/observability"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "github.com/MathTrail/mentor-api/docs"
)

func main() {
	// Load config early so observability can read its endpoints.
	cfg := config.Load()

	// --- Tracing ---
	tracerShutdown, err := observability.InitTracer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init tracer: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracerShutdown(ctx)
	}()

	// --- Metrics ---
	metricsShutdown, err := observability.InitMetrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init metrics: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsShutdown(ctx)
	}()

	// --- Profiling ---
	profiler, err := observability.InitPyroscope(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init pyroscope: %v\n", err)
		os.Exit(1)
	}
	defer profiler.Stop()

	// Initialize DI container (pool gets otelpgx tracer, router gets otelgin middleware)
	container := app.NewContainer()

	// Verify all dependencies are ready
	if !container.Ready() {
		container.Logger.Fatal("application container not ready")
	}

	container.Logger.Info("starting mentor-api server",
		zap.String("port", container.Config.ServerPort),
		zap.String("db_host", container.Config.DBHost),
		zap.String("otel_endpoint", cfg.OTelEndpoint),
		zap.String("pyroscope_endpoint", cfg.PyroscopeEndpoint),
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", container.Config.ServerPort),
		Handler: container.Router.(*gin.Engine),
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			container.Logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	container.Logger.Info("server started successfully")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	container.Logger.Info("shutting down server...")

	// Graceful shutdown with 15 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		container.Logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	container.Logger.Info("server shutdown complete")
	// deferred OTel flush and Pyroscope stop run here
}
