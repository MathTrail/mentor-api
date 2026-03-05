package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// mockService is a test double for roadmap.Service.
type mockService struct {
	fn func(ctx context.Context, studentID uuid.UUID) (*Recommendation, error)
}

func (m *mockService) GetRecommendations(ctx context.Context, studentID uuid.UUID) (*Recommendation, error) {
	if m.fn != nil {
		return m.fn(ctx, studentID)
	}
	return &Recommendation{
		StudentID:  studentID,
		FocusAreas: []string{"algebra"},
		Message:    "stub",
	}, nil
}

const (
	recommendationsPath = "/api/v1/roadmap/recommendations"
	userIDHeader        = "X-User-ID"
)

func testRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(recommendationsPath, h.GetRecommendations)
	return r
}

func TestGetRecommendationsSuccess(t *testing.T) {
	svc := &mockService{}
	hdl := NewHandler(svc, zap.NewNop())
	router := testRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(userIDHeader, uuid.New().String())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var rec Recommendation
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(rec.FocusAreas) == 0 {
		t.Error("expected at least one focus area")
	}
}

func TestGetRecommendationsMissingHeader(t *testing.T) {
	svc := &mockService{}
	hdl := NewHandler(svc, zap.NewNop())
	router := testRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "MISSING_USER_ID" {
		t.Errorf("code: got %q, want %q", body["code"], "MISSING_USER_ID")
	}
}

func TestGetRecommendationsInvalidUUID(t *testing.T) {
	svc := &mockService{}
	hdl := NewHandler(svc, zap.NewNop())
	router := testRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(userIDHeader, "not-a-uuid")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "INVALID_USER_ID" {
		t.Errorf("code: got %q, want %q", body["code"], "INVALID_USER_ID")
	}
}

func TestGetRecommendationsServiceError(t *testing.T) {
	svc := &mockService{
		fn: func(_ context.Context, _ uuid.UUID) (*Recommendation, error) {
			return nil, errors.New("service boom")
		},
	}
	hdl := NewHandler(svc, zap.NewNop())
	router := testRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(userIDHeader, uuid.New().String())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "INTERNAL_ERROR" {
		t.Errorf("code: got %q, want %q", body["code"], "INTERNAL_ERROR")
	}
}
