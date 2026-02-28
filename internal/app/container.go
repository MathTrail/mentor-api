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
func NewContainer(cfg *config.Config, logger *zap.Logger) (*Container, error) {
	// EnvPgPool uses credentials injected by VSO via the mentor-api-db-secret
	// K8s Secret. On lease renewal VSO triggers a rolling restart of this
	// Deployment, so no in-process credential refresh is needed.
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s user=%s password=%s",
		cfg.PgHost, cfg.PgPort, cfg.PgDatabase, cfg.PgSSLMode,
		cfg.PgUser, cfg.PgPassword,
	)
	db, err := postgres.NewEnvPgPool(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("env pg pool: %w", err)
	}

	// Verify the pool is reachable on startup.
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
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
