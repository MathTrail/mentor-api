package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"go.uber.org/zap"
)

// testConfig returns a minimal config suitable for router tests.
func testConfig() *config.Config {
	return &config.Config{SwaggerEnabled: true}
}

// --- test doubles ---

// mockDB implements postgres.DB for health check testing.
type mockDB struct {
	pingErr error
}

func (m *mockDB) Query(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error { return nil }
func (m *mockDB) Ping(_ context.Context) error                     { return m.pingErr }

// Compile-time interface check.
var _ postgres.DB = (*mockDB)(nil)

// mockFeedbackService implements feedback.Service for handler construction.
type mockFeedbackService struct{}

func (m *mockFeedbackService) ProcessFeedback(_ context.Context, req *feedback.FeedbackRequest) (*feedback.StrategyUpdate, error) {
	return &feedback.StrategyUpdate{StudentID: req.StudentID}, nil
}

// testRouter builds a router with stub handlers for use in tests.
func testRouter() (*feedback.Handler, *roadmap.Handler, *mockDB) {
	db := &mockDB{}
	feedbackHandler := feedback.NewHandler(&mockFeedbackService{}, zap.NewNop())
	roadmapHandler := roadmap.NewHandler(roadmap.NewService(zap.NewNop()), zap.NewNop())
	return feedbackHandler, roadmapHandler, db
}

// --- tests ---

func TestHealthStartup(t *testing.T) {
	fh, rh, db := testRouter()
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("startup: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthLiveness(t *testing.T) {
	fh, rh, db := testRouter()
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("liveness: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReadyOK(t *testing.T) {
	fh, rh, db := testRouter()
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReadyDBDown(t *testing.T) {
	fh, rh, _ := testRouter()
	db := &mockDB{pingErr: errors.New("connection refused")}
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready (db down): got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestRoadmapRecommendationsSuccess(t *testing.T) {
	fh, rh, db := testRouter()
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roadmap/recommendations", nil)
	req.Header.Set(middleware.UserIDHeader, "00000000-0000-0000-0000-000000000001")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("recommendations: got %d, want %d", w.Code, http.StatusOK)
	}

	var rec roadmap.Recommendation
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(rec.FocusAreas) == 0 {
		t.Error("expected at least one focus area")
	}
}

func TestRoadmapRecommendationsMissingHeader(t *testing.T) {
	fh, rh, db := testRouter()
	router := NewRouter(fh, rh, db, testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roadmap/recommendations", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing header: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}
