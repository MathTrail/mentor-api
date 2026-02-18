package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

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
	Pool   *pgxpool.Pool

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
	ctx := context.Background()

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logging.NewLogger(cfg.LogLevel)

	// Connect to database
	pool := database.NewPool(ctx, cfg, logger)

	// Initialize LLM client (mock for now, will be replaced with real implementation)
	llmClient := clients.NewMockLLMClient()

	// Initialize feedback components
	feedbackRepo := feedback.NewRepository(pool)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, logger)
	feedbackController := feedback.NewController(feedbackService, logger)

	// Create router
	router := server.NewRouter(feedbackController, pool, logger)

	return &Container{
		Config:             cfg,
		Logger:             logger,
		Pool:               pool,
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
		c.Pool != nil &&
		c.FeedbackRepository != nil &&
		c.FeedbackService != nil &&
		c.FeedbackController != nil &&
		c.Router != nil
}
