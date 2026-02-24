package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestUserSpanAttributes_SetsAttribute(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	tracer := tp.Tracer("test")

	r := gin.New()
	// Simulate otelgin — start a span so UserSpanAttributes has something to write to.
	r.Use(func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), "test-span")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(UserSpanAttributes())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "user-42")
	r.ServeHTTP(w, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	found := false
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "user.id" && attr.Value.AsString() == "user-42" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected user.id=user-42 attribute on span, got %v", spans[0].Attributes)
	}
}

func TestUserSpanAttributes_NoHeader(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	tracer := tp.Tracer("test")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), "test-span")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(UserSpanAttributes())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No X-User-ID header.
	r.ServeHTTP(w, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "user.id" {
			t.Errorf("expected no user.id attribute, but found %v", attr.Value.AsString())
		}
	}
}
