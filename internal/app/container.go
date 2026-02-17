package app

import (
	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/database"
	"github.com/MathTrail/mentor-api/internal/feedback"
	"github.com/MathTrail/mentor-api/internal/logging"
	"github.com/MathTrail/mentor-api/internal/server"
	"github.com/MathTrail/mentor-api/internal/strategy"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Container holds all application dependencies
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *gorm.DB

	// Clients
	ProfileClient clients.ProfileClient
	LLMClient     clients.LLMClient

	// Components
	Analyzer           *strategy.Analyzer
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

	// Connect to database
	db := database.NewConnection(cfg, logger)

	// Initialize mock clients (will be replaced with real implementations later)
	profileClient := clients.NewMockProfileClient()
	llmClient := clients.NewMockLLMClient()

	// Initialize strategy analyzer
	analyzer := strategy.NewAnalyzer()

	// Initialize feedback components
	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, profileClient, analyzer, logger)
	feedbackController := feedback.NewController(feedbackService, logger)

	// Create router
	router := server.NewRouter(feedbackController, logger)

	return &Container{
		Config:             cfg,
		Logger:             logger,
		DB:                 db,
		ProfileClient:      profileClient,
		LLMClient:          llmClient,
		Analyzer:           analyzer,
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
