package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSanitizeQueryMasksSensitiveKeys(t *testing.T) {
	raw := "token=REDACTED&page=1&api_key=REDACTED"
	got := sanitizeQuery(raw)

	// Verify sensitive keys are masked.
	for _, key := range []string{"token", "api_key"} {
		if !contains(got, key+"=%2A%2A%2A") && !contains(got, key+"=***") {
			t.Errorf("expected %s to be masked in %q", key, got)
		}
	}
	// Verify non-sensitive key is preserved.
	if !contains(got, "page=1") {
		t.Errorf("expected page=1 in %q", got)
	}
}

func TestSanitizeQueryEmpty(t *testing.T) {
	if got := sanitizeQuery(""); got != "" {
		t.Errorf("sanitizeQuery(\"\") = %q, want \"\"", got)
	}
}

func TestSanitizeQueryNoSensitiveKeys(t *testing.T) {
	raw := "page=1&limit=10"
	got := sanitizeQuery(raw)
	if !contains(got, "page=1") || !contains(got, "limit=10") {
		t.Errorf("expected all params preserved in %q", got)
	}
}

func TestSanitizeQueryCaseInsensitive(t *testing.T) {
	raw := "TOKEN=test-value&Password=test-value"
	got := sanitizeQuery(raw)
	for _, key := range []string{"TOKEN", "Password"} {
		if !contains(got, key+"=%2A%2A%2A") && !contains(got, key+"=***") {
			t.Errorf("expected %s to be masked in %q", key, got)
		}
	}
}

func TestIsInternalPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/health/ready", true},
		{"/health/liveness", true},
		{"/metrics", true},
		{"/api/v1/feedback", false},
		{"/swagger/index.html", false},
	}
	for _, tt := range tests {
		if got := isInternalPath(tt.path); got != tt.want {
			t.Errorf("isInternalPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// contains checks if substr is in s (simple helper to avoid importing strings in tests).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// newObservedLogger returns a zap.Logger that captures log entries for assertions.
func newObservedLogger(level zapcore.Level) (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	return zap.New(core), logs
}

// ginRouter builds a test router with ZapLogger and a handler that writes statusCode.
func ginRouter(logger *zap.Logger, path string, statusCode int) *gin.Engine {
	r := gin.New()
	r.Use(ZapLogger(logger))
	r.GET(path, func(c *gin.Context) {
		c.Status(statusCode)
	})
	return r
}

func TestZapLogger_API_200_IsLogged(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.InfoLevel)
	r := ginRouter(logger, "/api/v1/feedback", http.StatusOK)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/feedback", nil))

	if logs.Len() != 1 {
		t.Errorf("expected 1 log entry, got %d", logs.Len())
	}
	if logs.All()[0].Level != zap.InfoLevel {
		t.Errorf("expected Info level, got %v", logs.All()[0].Level)
	}
}

func TestZapLogger_Health_200_IsSkipped(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.DebugLevel)
	r := ginRouter(logger, "/health/ready", http.StatusOK)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if logs.Len() != 0 {
		t.Errorf("expected 0 log entries for health 200, got %d", logs.Len())
	}
}

func TestZapLogger_Health_500_IsLogged(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.ErrorLevel)
	r := ginRouter(logger, "/health/ready", http.StatusInternalServerError)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if logs.Len() != 1 {
		t.Errorf("expected 1 log entry for health 500, got %d", logs.Len())
	}
	if logs.All()[0].Level != zap.ErrorLevel {
		t.Errorf("expected Error level, got %v", logs.All()[0].Level)
	}
}

func TestZapLogger_Metrics_200_IsSkipped(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.DebugLevel)
	r := ginRouter(logger, "/metrics", http.StatusOK)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if logs.Len() != 0 {
		t.Errorf("expected 0 log entries for /metrics 200, got %d", logs.Len())
	}
}

func TestZapLogger_4xx_IsWarn(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.WarnLevel)
	r := ginRouter(logger, "/test", http.StatusBadRequest)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	if logs.Len() != 1 {
		t.Errorf("expected 1 log entry, got %d", logs.Len())
	}
	if logs.All()[0].Level != zap.WarnLevel {
		t.Errorf("expected Warn level, got %v", logs.All()[0].Level)
	}
}

func TestZapLogger_5xx_IsError(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.ErrorLevel)
	r := ginRouter(logger, "/test", http.StatusInternalServerError)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	if logs.Len() != 1 {
		t.Errorf("expected 1 log entry, got %d", logs.Len())
	}
	if logs.All()[0].Level != zap.ErrorLevel {
		t.Errorf("expected Error level, got %v", logs.All()[0].Level)
	}
}

func TestZapLogger_RequestIDField(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.InfoLevel)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(RequestIDKey, "req-abc-123")
		c.Next()
	})
	router.Use(ZapLogger(logger))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}
	entry := logs.All()[0]
	found := false
	for _, f := range entry.Context {
		if f.Key == "request_id" && f.String == "req-abc-123" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected request_id field in log entry")
	}
}

func TestZapLogger_QuerySanitized(t *testing.T) {
	logger, logs := newObservedLogger(zapcore.InfoLevel)

	router := gin.New()
	router.Use(ZapLogger(logger))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?token=secret&page=1", nil)
	router.ServeHTTP(w, req)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}
	entry := logs.All()[0]
	for _, f := range entry.Context {
		if f.Key == "query" && contains(f.String, "secret") {
			t.Errorf("sensitive value leaked in query log: %q", f.String)
		}
	}
}
