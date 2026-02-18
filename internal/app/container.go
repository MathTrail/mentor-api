package app

import (
	"context"
	"fmt"

	dapr "github.com/dapr/go-sdk/client"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/database"
	"github.com/MathTrail/mentor-api/internal/feedback"
	"github.com/MathTrail/mentor-api/internal/logging"
	"github.com/MathTrail/mentor-api/internal/server"
	"go.uber.org/zap"
)

// Container holds all application dependencies
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *database.DaprDB

	// Clients
	LLMClient clients.LLMClient

	// Components
	FeedbackRepository feedback.Repository
	FeedbackService    feedback.Service
	FeedbackController *feedback.Controller

	// Server
	Router interface{}
}

// NewContainer creates and wires all application dependencies
func NewContainer() *Container {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logging.NewLogger(cfg.LogLevel)

	// Connect to the Dapr sidecar. NewClient() reads DAPR_GRPC_PORT from the environment
	// (injected automatically by the Dapr sidecar into the application container).
	daprClient, err := dapr.NewClient()
	if err != nil {
		panic(fmt.Sprintf("failed to create Dapr client: %v", err))
	}

	// DaprDB wraps the Dapr binding so the app never handles DB credentials directly.
	db := database.NewDaprDB(daprClient, cfg.DBBindingName)

	// Verify the binding is reachable on startup.
	if err := db.Ping(context.Background()); err != nil {
		logger.Fatal("database binding not reachable", zap.Error(err))
	}

	// Initialize LLM client (mock for now, will be replaced with real implementation)
	llmClient := clients.NewMockLLMClient()

	// Initialize feedback components
	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, logger)
	feedbackController := feedback.NewController(feedbackService, logger)

	// Create router
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
	}
}

// Ready returns true if all dependencies are initialized
func (c *Container) Ready() bool {
	return c.Config != nil &&
		c.Logger != nil &&
		c.DB != nil &&
		c.FeedbackRepository != nil &&
		c.FeedbackService != nil &&
		c.FeedbackController != nil &&
		c.Router != nil
}
