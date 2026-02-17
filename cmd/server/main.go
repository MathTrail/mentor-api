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
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize DI container
	container := app.NewContainer()

	// Verify all dependencies are ready
	if !container.Ready() {
		container.Logger.Fatal("application container not ready")
	}

	container.Logger.Info("starting mentor-api server",
		zap.String("port", container.Config.ServerPort),
		zap.String("db_host", container.Config.DBHost),
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
}
