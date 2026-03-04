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

// Container holds the dependencies consumed by the Server.
// Internal wiring is kept as local, variables in NewContainer and is not exposed.
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	Router *gin.Engine
	stop   func()
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully.
func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	db, err := initDB(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	llmClient := clients.NewLLMClient()

	feedbackRepo := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepo, llmClient, cfg.LLMTimeout, logger)
	feedbackHandler := feedback.NewHandler(feedbackService, logger)

	roadmapService := roadmap.NewService(logger)
	roadmapHandler := roadmap.NewHandler(roadmapService, logger)

	router := httpserver.NewRouter(feedbackHandler, roadmapHandler, db, cfg, logger)

	return &Container{
		Config: cfg,
		Logger: logger,
		Router: router,
		stop:   db.Close, // captures *DynamicPool — no DB interface change needed
	}, nil
}

// Close releases resources held by the container.
// Call once after the HTTP server has stopped accepting requests.
func (c *Container) Close() {
	c.stop()
}

// initDB builds the DSN and creates a DynamicPool that rotates credentials
// in-process when VSO-mounted Secret files change.
func initDB(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*postgres.DynamicPool, error) {
	baseDSN := fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		cfg.PgHost, cfg.PgPort, cfg.PgDatabase, cfg.PgSSLMode,
	)
	db, err := postgres.NewDynamicPool(ctx, baseDSN, cfg.PgCredentialsDir, logger)
	if err != nil {
		return nil, fmt.Errorf("dynamic pg pool: %w", err)
	}
	return db, nil
}
