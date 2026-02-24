package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserSpanAttributes reads the X-User-ID header injected by Oathkeeper and sets
// it as an attribute on the active OTel span. This allows filtering traces by
// authenticated user in Grafana Tempo.
func UserSpanAttributes() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			span := trace.SpanFromContext(c.Request.Context())
			span.SetAttributes(attribute.String("user.id", userID))
		}
		c.Next()
	}
}
