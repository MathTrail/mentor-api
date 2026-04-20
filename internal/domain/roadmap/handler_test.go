package roadmap_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	roadmapmocks "github.com/MathTrail/mentor-api/internal/domain/roadmap/mocks"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	recommendationsPath = "/api/v1/roadmap/recommendations"
	statusFmt           = "status: got %d, want %d"
	codeFmt             = "code: got %q, want %q"
)

func testRoadmapRouter(h *roadmap.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(recommendationsPath, h.GetRecommendations)
	return r
}

func TestGetRecommendationsSuccess(t *testing.T) {
	svc := roadmapmocks.NewMockService(t)
	svc.EXPECT().GetRecommendations(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, studentID uuid.UUID) (*roadmap.Recommendation, error) {
			return &roadmap.Recommendation{
				StudentID:  studentID,
				FocusAreas: []string{"algebra"},
				Message:    "stub",
			}, nil
		})

	hdl := roadmap.NewHandler(svc, zap.NewNop())
	router := testRoadmapRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(middleware.UserIDHeader, uuid.New().String())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf(statusFmt, w.Code, http.StatusOK)
	}

	var rec roadmap.Recommendation
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(rec.FocusAreas) == 0 {
		t.Error("expected at least one focus area")
	}
}

func TestGetRecommendationsMissingHeader(t *testing.T) {
	svc := roadmapmocks.NewMockService(t)
	hdl := roadmap.NewHandler(svc, zap.NewNop())
	router := testRoadmapRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf(statusFmt, w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "MISSING_USER_ID" {
		t.Errorf(codeFmt, body["code"], "MISSING_USER_ID")
	}
}

func TestGetRecommendationsInvalidUUID(t *testing.T) {
	svc := roadmapmocks.NewMockService(t)
	hdl := roadmap.NewHandler(svc, zap.NewNop())
	router := testRoadmapRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(middleware.UserIDHeader, "not-a-uuid")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf(statusFmt, w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "INVALID_USER_ID" {
		t.Errorf(codeFmt, body["code"], "INVALID_USER_ID")
	}
}

func TestGetRecommendationsServiceError(t *testing.T) {
	svc := roadmapmocks.NewMockService(t)
	svc.EXPECT().GetRecommendations(mock.Anything, mock.Anything).
		Return(nil, errors.New("service boom"))

	hdl := roadmap.NewHandler(svc, zap.NewNop())
	router := testRoadmapRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, recommendationsPath, nil)
	req.Header.Set(middleware.UserIDHeader, uuid.New().String())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf(statusFmt, w.Code, http.StatusInternalServerError)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "INTERNAL_ERROR" {
		t.Errorf(codeFmt, body["code"], "INTERNAL_ERROR")
	}
}
