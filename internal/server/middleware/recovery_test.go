package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/apierror"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestZapRecovery_ReturnsStructuredJSON(t *testing.T) {
	r := gin.New()
	r.Use(ZapRecovery(zap.NewNop()))
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp apierror.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp.Code != "INTERNAL_ERROR" {
		t.Errorf("code = %q, want %q", resp.Code, "INTERNAL_ERROR")
	}
	if resp.Message != "an unexpected error occurred" {
		t.Errorf("message = %q, want %q", resp.Message, "an unexpected error occurred")
	}
}
