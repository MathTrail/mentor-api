package server

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// RequestIDHeader is the HTTP header for request correlation.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the Gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestID injects a unique request ID into every request.
// If the client sends X-Request-ID, it is reused; otherwise a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// ZapLogger returns a Gin middleware that logs every HTTP request using zap.
// Each log line includes method, path, status, latency, client IP, and request ID.
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()

		// Skip logging for successful health probes and Dapr sidecar calls.
		if status < 400 && isInternalPath(path) {
			return
		}

		latency := time.Since(start)

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		}

		if id, ok := c.Get(RequestIDKey); ok {
			fields = append(fields, zap.String("request_id", id.(string)))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		switch {
		case status >= 500:
			logger.Error("request", fields...)
		case status >= 400:
			logger.Warn("request", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}

// ZapRecovery returns a Gin middleware that recovers from panics and logs them with zap.
func ZapRecovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		logger.Error("panic recovered",
			zap.Any("error", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		)
		c.AbortWithStatus(500)
	})
}

// internalPrefixes lists path prefixes that are only logged on errors.
var internalPrefixes = []string{"/health/", "/dapr/"}

// isInternalPath reports whether the path matches a probe or sidecar prefix.
func isInternalPath(path string) bool {
	for _, p := range internalPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
