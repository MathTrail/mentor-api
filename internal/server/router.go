package server

import (
	"github.com/MathTrail/mentor-api/internal/feedback"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter creates and configures the Gin router with all routes and middleware
func NewRouter(feedbackController *feedback.Controller, logger *zap.Logger) *gin.Engine {
	// Set Gin to release mode (disable debug logs)
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(RequestID())
	router.Use(ZapLogger(logger))
	router.Use(ZapRecovery(logger))

	// Health check endpoints (for Kubernetes probes)
	router.GET("/health/startup", healthStartup)
	router.GET("/health/liveness", healthLiveness)
	router.GET("/health/ready", healthReady)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/feedback", feedbackController.SubmitFeedback)
	}

	return router
}

// healthStartup indicates that the application has started
func healthStartup(c *gin.Context) {
	c.JSON(200, gin.H{"status": "started"})
}

// healthLiveness indicates that the application is running
func healthLiveness(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// healthReady indicates that the application is ready to serve traffic
func healthReady(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ready"})
}
