package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const testClientID = "client-id-123"

func init() { gin.SetMode(gin.TestMode) }

func TestRequestIDClientProvided(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(RequestIDKey))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, testClientID)
	r.ServeHTTP(w, req)

	if w.Body.String() != testClientID {
		t.Errorf("body = %q, want %q", w.Body.String(), testClientID)
	}
	if got := w.Header().Get(RequestIDHeader); got != testClientID {
		t.Errorf("response header = %q, want %q", got, testClientID)
	}
}

func TestRequestIDTraceIDFallback(t *testing.T) {
	// Create a real tracer that generates valid TraceIDs.
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	tracer := tp.Tracer("test")

	r := gin.New()
	// Simulate otelgin by starting a span before RequestID runs.
	r.Use(func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), "test-span")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(RequestIDKey))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	got := w.Body.String()
	// TraceID is a 32-character hex string.
	if len(got) != 32 {
		t.Errorf("expected 32-char TraceID, got %q (len %d)", got, len(got))
	}
	if got != w.Header().Get(RequestIDHeader) {
		t.Errorf("header %q != body %q", w.Header().Get(RequestIDHeader), got)
	}
}

func TestRequestIDUUIDFallback(t *testing.T) {
	r := gin.New()
	// No otelgin — span context has no TraceID.
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(RequestIDKey))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	got := w.Body.String()
	// UUID v4 format: 8-4-4-4-12 = 36 chars.
	if len(got) != 36 {
		t.Errorf("expected 36-char UUID, got %q (len %d)", got, len(got))
	}

	// Verify no valid TraceID was present.
	span := trace.SpanFromContext(context.Background())
	if span.SpanContext().HasTraceID() {
		t.Error("background context should not have a TraceID")
	}
}
