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
	daprClient, err := connectDapr(logger)
	if err != nil {
		return nil, fmt.Errorf("dapr client: %w", err)
	}

	// DaprDB wraps the Dapr binding so the app never handles DB credentials directly.
	db := database.NewDaprDB(daprClient, cfg.DBBindingName)

	// Verify the binding is reachable on startup.
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database binding not reachable: %w", err)
	}

	// Initialize LLM client (mock for now, will be replaced with real implementation).
	llmClient := clients.NewMockLLMClient()

	// Initialize feedback components.
	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, logger)
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
func connectDapr(logger *zap.Logger) (dapr.Client, error) {
	var (
		daprClient dapr.Client
		err        error
	)
	for attempt := 1; attempt <= 10; attempt++ {
		daprClient, err = dapr.NewClient()
		if err == nil {
			return daprClient, nil
		}
		logger.Warn("dapr sidecar not ready, retrying",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return nil, fmt.Errorf("failed after 10 attempts: %w", err)
}
