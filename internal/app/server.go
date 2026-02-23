package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Server wraps the HTTP server with graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer creates a Server from the wired Container.
func NewServer(container *Container) *Server {
	return &Server{
		logger: container.Logger,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", container.Config.ServerPort),
			Handler: container.Router,
		},
	}
}

// Run starts the HTTP listener and blocks until SIGINT/SIGTERM is received.
// It then performs a graceful shutdown with a 15-second timeout.
func (s *Server) Run() error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("listening", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case sig := <-quit:
		s.logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("server shut down gracefully")
	return nil
}
