package app

import (
	"context"
	"fmt"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/gin-gonic/gin"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/database"
	"github.com/MathTrail/mentor-api/internal/feedback"
	"github.com/MathTrail/mentor-api/internal/server"
	"go.uber.org/zap"
)

// Container holds all application dependencies.
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	DB     database.DB

	// Clients
	LLMClient clients.LLMClient

	// Components
	FeedbackRepository feedback.Repository
	FeedbackService    feedback.Service
	FeedbackController *feedback.Controller

	// Server
	Router *gin.Engine
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully (e.g. flush observability before exit).
func NewContainer(cfg *config.Config, logger *zap.Logger) (*Container, error) {
	// Connect to the Dapr sidecar with retry.
	daprClient, err := connectDapr(context.Background(), logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("dapr client: %w", err)
	}

	// DaprDB wraps the Dapr binding so the app never handles DB credentials directly.
	db := database.NewDaprDB(daprClient, cfg.DBBindingName)

	// Verify the binding is reachable on startup.
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database binding not reachable: %w", err)
	}

	// Initialize LLM client.
	llmClient := clients.NewLLMClient()

	// Initialize feedback components.
	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, cfg.LLMTimeout, logger)
	feedbackController := feedback.NewController(feedbackService, logger)

	// Create router.
	router := server.NewRouter(feedbackController, db, logger)

	return &Container{
		Config:             cfg,
		Logger:             logger,
		DB:                 db,
		LLMClient:          llmClient,
		FeedbackRepository: feedbackRepo,
		FeedbackService:    feedbackService,
		FeedbackController: feedbackController,
		Router:             router,
	}, nil
}

// connectDapr attempts to connect to the Dapr sidecar with linear backoff.
// The sidecar and the app container start concurrently in Kubernetes,
// so the sidecar may not be ready immediately.
func connectDapr(ctx context.Context, logger *zap.Logger, cfg *config.Config) (dapr.Client, error) {
	var err error
	for attempt := 1; attempt <= cfg.DaprMaxRetries; attempt++ {
		client, connErr := dapr.NewClient()
		if connErr == nil {
			return client, nil
		}
		err = connErr

		logger.Warn("dapr sidecar not ready, retrying",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		// Wait for backoff or context cancellation.
		select {
		case <-time.After(time.Duration(attempt) * time.Second):
			// continue
		case <-ctx.Done():
			return nil, fmt.Errorf("interrupted during connection: %w", ctx.Err())
		}
	}
	return nil, fmt.Errorf("dapr unavailable after %d attempts: %w", cfg.DaprMaxRetries, err)
}
