package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathTrail/mentor-api/internal/infra/postgres"
)

// HealthHandler serves Kubernetes health probe endpoints.
type HealthHandler struct {
	db postgres.DB
}

// NewHealthHandler creates a HealthHandler backed by the given DB pool.
func NewHealthHandler(db postgres.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Startup indicates that the application has started.
func (h *HealthHandler) Startup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// Live indicates that the application is running.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready verifies DB connectivity before reporting ready.
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.db.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "db: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
