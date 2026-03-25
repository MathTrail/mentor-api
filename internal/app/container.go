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

	onboardingConsumer, kafkaClient, err := initOnboardingConsumer(cfg, db, logger)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Container{
		Config:             cfg,
		Logger:             logger,
		Router:             router,
		OnboardingConsumer: onboardingConsumer,
		// Closes resources in reverse init order. kafkaClient.Close() is
		// idempotent — safe even if Consumer.Start() already closed it.
		stop: func() {
			kafkaClient.Close()
			db.Close()
		},
	}, nil
}

// RunWorkers starts all background consumers and blocks until ctx is cancelled
// or a worker returns an error. Intended to be run via errgroup alongside the
// HTTP server so that container.Close() is only called after all workers exit.
func (c *Container) RunWorkers(ctx context.Context) error {
	return c.OnboardingConsumer.Start(ctx)
}

// Close releases resources held by the container.
// Call once after RunWorkers and the HTTP server have both stopped.
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
		SASLUsername:     cfg.KafkaSASLUsername,
		SASLPassword:     cfg.KafkaSASLPassword,
	})
}

// initOnboardingConsumer wires the Kafka consumer that reads
// students.onboarding.ready events and persists recommendations.
// It returns the Consumer and the underlying *kgo.Client so the
// caller can manage the client lifecycle via Container.stop.
func initOnboardingConsumer(cfg *config.Config, db *postgres.DynamicPool, logger *zap.Logger) (*onboarding.Consumer, *kgo.Client, error) {
	// TopicValidator fails fast at startup if Apicurio is unreachable or the
	// subject does not exist — ensures schema alignment before any message is consumed.
	validator, err := kafkainfra.NewValidator(cfg.ApicurioURL, cfg.OnboardingSchemaSubject)
	if err != nil {
		return nil, nil, fmt.Errorf("schema validator init: %w", err)
	}

	kafkaClient, err := initKafkaClient(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("kafka client: %w", err)
	}

	repo := onboarding.NewRepository(db)
	consumer := onboarding.NewConsumer(kafkaClient, repo, validator, logger)
	return consumer, kafkaClient, nil
}
