package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/onboarding"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	"github.com/MathTrail/mentor-api/internal/runner"
	httpserver "github.com/MathTrail/mentor-api/internal/transport/http"
	"go.uber.org/zap"
)

// Container holds the dependencies consumed by the Server.
// Internal wiring is kept as local variables in NewContainer and is not exposed.
type Container struct {
	Config  *config.Config
	Logger  *zap.Logger
	Router  *gin.Engine
	Workers []runner.Worker
	stop    func()
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully.
func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	db, err := postgres.NewDynamicPool(ctx, cfg.PostgresDSN(), cfg.PgCredentialsDir, logger)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	// TopicValidator fails fast at startup if Apicurio is unreachable or the
	// subject does not exist — ensures schema alignment before any message is consumed.
	validator, err := kafkainfra.NewValidator(cfg.ApicurioURL, cfg.OnboardingSchemaSubject)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("schema validator: %w", err)
	}

	kafkaClient, err := kafkainfra.NewClientFromConfig(cfg, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("kafka client: %w", err)
	}

	llmClient := clients.NewFeedbackClient()

	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, cfg.LLMTimeout, logger)
	feedbackHandler := feedback.NewHandler(feedbackService, logger)

	roadmapService := roadmap.NewService(logger)
	roadmapHandler := roadmap.NewHandler(roadmapService, logger)

	healthHandler := httpserver.NewHealthHandler(db)
	router := httpserver.NewRouter(feedbackHandler, roadmapHandler, healthHandler, cfg, logger)

	onboardingRepo := onboarding.NewRepository(db)
	onboardingConsumer := onboarding.NewConsumer(kafkaClient, onboardingRepo, validator, logger)

	return &Container{
		Config:  cfg,
		Logger:  logger,
		Router:  router,
		Workers: []runner.Worker{onboardingConsumer},
		// Closes resources in reverse init order.
		stop: func() {
			kafkaClient.Close()
			db.Close()
		},
	}, nil
}

// Close releases resources held by the container.
// Call once after RunWorkers and the HTTP server have both stopped.
func (c *Container) Close() {
	c.stop()
}
