package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/MathTrail/mentor-api/internal/feedback"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter creates and configures the Gin router with all routes and middleware
func NewRouter(feedbackController *feedback.Controller, pool *pgxpool.Pool, logger *zap.Logger) *gin.Engine {
	// Set Gin to release mode (disable debug logs)
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(otelgin.Middleware("mentor-api")) // extracts traceparent from Dapr, creates child spans
	router.Use(UserSpanAttributes())             // injects X-User-ID (from Oathkeeper) into active OTel span
	router.Use(RequestID())
	router.Use(ZapLogger(logger))
	router.Use(ZapRecovery(logger))

	// Observability endpoints
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoints (for Kubernetes probes)
	router.GET("/health/startup", healthStartup)
	router.GET("/health/liveness", healthLiveness)
	router.GET("/health/ready", healthReady(pool))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/feedback", feedbackController.SubmitFeedback)
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

// healthReady verifies that all dependencies are available before reporting ready
func healthReady(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "db: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
