package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	httpserver "github.com/MathTrail/mentor-api/internal/transport/http"
	"go.uber.org/zap"
)

// Container holds all application dependencies.
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	DB     postgres.DB

	// Clients
	LLMClient clients.LLMClient

	// Feedback components
	FeedbackRepository feedback.Repository
	FeedbackService    feedback.Service
	FeedbackHandler    *feedback.Handler

	// Roadmap components
	RoadmapService roadmap.Service
	RoadmapHandler *roadmap.Handler

	// Server
	Router *gin.Engine
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully (e.g. flush observability before exit).
func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	// DynamicPool reads credentials from VSO-mounted Secret files (cfg.PgCredentialsDir)
	// and rotates the pool in-process when files change — no pod restart needed.
	baseDSN := fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		cfg.PgHost, cfg.PgPort, cfg.PgDatabase, cfg.PgSSLMode,
	)
	db, err := postgres.NewDynamicPool(ctx, baseDSN, cfg.PgCredentialsDir, logger)
	if err != nil {
		return nil, fmt.Errorf("dynamic pg pool: %w", err)
	}

	// Initialize LLM client.
	llmClient := clients.NewLLMClient()

	// Initialize feedback components.
	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, cfg.LLMTimeout, logger)
	feedbackHandler := feedback.NewHandler(feedbackService, logger)

	// Initialize roadmap components.
	roadmapService := roadmap.NewService(logger)
	roadmapHandler := roadmap.NewHandler(roadmapService, logger)

	// Create router.
	router := httpserver.NewRouter(feedbackHandler, roadmapHandler, db, cfg, logger)

	return &Container{
		Config:             cfg,
		Logger:             logger,
		DB:                 db,
		LLMClient:          llmClient,
		FeedbackRepository: feedbackRepo,
		FeedbackService:    feedbackService,
		FeedbackHandler:    feedbackHandler,
		RoadmapService:     roadmapService,
		RoadmapHandler:     roadmapHandler,
		Router:             router,
	}, nil
}
