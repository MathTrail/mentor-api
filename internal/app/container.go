package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/onboarding"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	httpserver "github.com/MathTrail/mentor-api/internal/transport/http"
	"go.uber.org/zap"
)

// Container holds the dependencies consumed by the Server.
// Internal wiring is kept as local, variables in NewContainer and is not exposed.
type Container struct {
	Config             *config.Config
	Logger             *zap.Logger
	Router             *gin.Engine
	OnboardingConsumer *onboarding.Consumer
	stop               func()
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully.
func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	db, err := initDB(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	llmClient := clients.NewFeedbackClient()

	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, cfg.LLMTimeout, logger)
	feedbackHandler := feedback.NewHandler(feedbackService, logger)

	roadmapService := roadmap.NewService(logger)
	roadmapHandler := roadmap.NewHandler(roadmapService, logger)

	router := httpserver.NewRouter(feedbackHandler, roadmapHandler, db, cfg, logger)

	// Kafka consumer: students.onboarding.ready → recommendations table
	kafkaClient, err := initKafkaClient(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("kafka client: %w", err)
	}
	onboardingRepo := onboarding.NewRepository(db)
	onboardingConsumer := onboarding.NewConsumer(kafkaClient, onboardingRepo, logger)

	return &Container{
		Config:             cfg,
		Logger:             logger,
		Router:             router,
		OnboardingConsumer: onboardingConsumer,
		stop:               db.Close,
	}, nil
}

// Close releases resources held by the container.
// Call once after the HTTP server has stopped accepting requests.
func (c *Container) Close() {
	c.stop()
}

// initDB creates a DynamicPool that rotates credentials
// in-process when VSO-mounted Secret files change.
func initDB(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*postgres.DynamicPool, error) {
	db, err := postgres.NewDynamicPool(ctx, cfg.PostgresDSN(), cfg.PgCredentialsDir, logger)
	if err != nil {
		return nil, fmt.Errorf("dynamic pg pool: %w", err)
	}
	return db, nil
}

// initKafkaClient creates a franz-go Kafka client for the onboarding consumer.
func initKafkaClient(cfg *config.Config) (*kgo.Client, error) {
	brokers := strings.Split(cfg.KafkaBootstrapServers, ",")
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "mentor-api-local"
	}
	return kafkainfra.NewClient(kafkainfra.ClientConfig{
		BootstrapServers: brokers,
		ConsumerGroup:    cfg.KafkaConsumerGroup,
		InstanceID:       podName,
		TLSCertDir:       cfg.KafkaCertDir,
	})
}
