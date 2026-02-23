package middleware

import (
	"net/http"

	"github.com/MathTrail/mentor-api/internal/apierror"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapRecovery returns a Gin middleware that recovers from panics and logs them
// with zap. It returns a structured JSON error response so clients can handle
// the failure gracefully.
func ZapRecovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		logger.Error("panic recovered",
			zap.Any("error", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.Response{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
	})
}
