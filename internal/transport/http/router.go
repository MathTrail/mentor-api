package httpserver

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter creates and configures the Gin router with all routes and middleware.
func NewRouter(
	feedbackHandler *feedback.Handler,
	roadmapHandler *roadmap.Handler,
	db postgres.DB,
	cfg *config.Config,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin to release mode (disable debug logs)
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware.
	// Order matters: otelgin wraps everything for tracing, ZapRecovery catches
	// panics from all downstream middleware and handlers.
	router.Use(otelgin.Middleware("mentor-api")) // extracts traceparent from Dapr, creates child spans
	router.Use(middleware.ZapRecovery(logger))   // must be early to catch panics in middleware below
	router.Use(middleware.UserSpanAttributes())  // injects X-User-ID (from Oathkeeper) into active OTel span
	router.Use(middleware.RequestID())           // links to OTel TraceID when X-Request-ID is absent
	router.Use(middleware.ZapLogger(logger))

	// Dapr app configuration endpoint.
	// The Dapr sidecar probes this on startup to discover pub/sub subscriptions.
	// Returning 200 with an empty object signals "no subscriptions" and suppresses sidecar 404 noise.
	router.GET("/dapr/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Observability endpoints
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoints (for Kubernetes probes)
	router.GET("/health/startup", healthStartup)
	router.GET("/health/liveness", healthLiveness)
	router.GET("/health/ready", healthReady(db))

	// Swagger UI (disabled in production via SWAGGER_ENABLED=false)
	if cfg.SwaggerEnabled {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/feedback", feedbackHandler.SubmitFeedback)
		v1.GET("/roadmap/recommendations", roadmapHandler.GetRecommendations)
	}

	return router
}

// healthStartup indicates that the application has started
func healthStartup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// healthLiveness indicates that the application is running
func healthLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// healthReady verifies DB connectivity via the Dapr binding before reporting ready.
func healthReady(db postgres.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "db: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
