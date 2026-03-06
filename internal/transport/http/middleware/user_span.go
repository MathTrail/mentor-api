package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserIDHeader is the HTTP header carrying the authenticated student ID,
// injected by Oathkeeper before the request reaches the service.
const UserIDHeader = "X-User-ID"

// UserSpanAttributes reads the X-User-ID header injected by Oathkeeper and sets
// it as an attribute on the active OTel span. This allows filtering traces by
// authenticated user in Grafana Tempo.
func UserSpanAttributes() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID := c.GetHeader(UserIDHeader); userID != "" {
			span := trace.SpanFromContext(c.Request.Context())
			span.SetAttributes(attribute.String("user.id", userID))
		}
		c.Next()
	}
}
