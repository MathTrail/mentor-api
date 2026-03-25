package httpserver

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter creates and configures the Gin router with all routes and middleware.
func NewRouter(
	feedbackHandler *feedback.Handler,
	roadmapHandler *roadmap.Handler,
	healthHandler *HealthHandler,
	cfg *config.Config,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin to release mode (disable debug logs)
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware.
	// Order matters: otelgin wraps everything for tracing, ZapRecovery catches
	// panics from all downstream middleware and handlers.
	router.Use(otelgin.Middleware("mentor-api")) // extracts W3C traceparent, creates child spans
	router.Use(middleware.ZapRecovery(logger))   // must be early to catch panics in middleware below
	router.Use(middleware.UserSpanAttributes())  // injects X-User-ID (from Oathkeeper) into active OTel span
	router.Use(middleware.RequestID())           // links to OTel TraceID when X-Request-ID is absent
	router.Use(middleware.ZapLogger(logger))

	// Observability endpoints
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoints (for Kubernetes probes)
	router.GET("/health/startup", healthHandler.Startup)
	router.GET("/health/liveness", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)

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
