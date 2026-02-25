package app

import (
	"context"
	"fmt"
	"time"

	dapr "github.com/dapr/go-sdk/client"
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
	// Connect to the Dapr sidecar with retry.
	daprClient, err := connectDapr(context.Background(), logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("dapr client: %w", err)
	}

	// DaprPgPool fetches dynamic DB credentials from Vault via the Dapr sidecar
	// ("vault-db" secret store component) and manages a pgxpool.Pool.
	// Credentials never touch etcd — they live only in the sidecar and this pool.
	pgDSNTpl := fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		cfg.PgHost, cfg.PgPort, cfg.PgDatabase, cfg.PgSSLMode,
	)
	db, err := postgres.NewDaprPgPool(
		context.Background(),
		daprClient,
		cfg.DBSecretStore,
		cfg.DBSecretKey,
		pgDSNTpl,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("vault pg pool: %w", err)
	}

	// Verify the pool is reachable on startup.
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
	}

	// Refresh credentials 10 minutes before the Vault default_ttl (1h).
	// Each refresh creates a new Vault lease and swaps the pool atomically.
	db.StartRefresh(context.Background(), 50*time.Minute)

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
