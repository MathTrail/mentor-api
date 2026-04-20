package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	feedbackmocks "github.com/MathTrail/mentor-api/internal/domain/feedback/mocks"
	"github.com/MathTrail/mentor-api/internal/domain/roadmap"
	pgmocks "github.com/MathTrail/mentor-api/internal/infra/postgres/mocks"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// testConfig returns a minimal config suitable for router tests.
func testConfig() *config.Config {
	return &config.Config{SwaggerEnabled: true}
}

// testDeps creates stub handlers and a MockDB for use in router tests.
// Callers must set up db.EXPECT().Ping(...) before hitting /health/ready.
func testDeps(t *testing.T) (*feedback.Handler, *roadmap.Handler, *pgmocks.MockDB) {
	t.Helper()
	db := pgmocks.NewMockDB(t)
	svc := feedbackmocks.NewMockService(t)
	feedbackHandler := feedback.NewHandler(svc, zap.NewNop())
	roadmapHandler := roadmap.NewHandler(roadmap.NewService(zap.NewNop()), zap.NewNop())
	return feedbackHandler, roadmapHandler, db
}

// --- tests ---

func TestHealthStartup(t *testing.T) {
	fh, rh, db := testDeps(t)
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("startup: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthLiveness(t *testing.T) {
	fh, rh, db := testDeps(t)
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("liveness: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReadyOK(t *testing.T) {
	fh, rh, db := testDeps(t)
	db.EXPECT().Ping(mock.Anything).Return(nil)
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReadyDBDown(t *testing.T) {
	fh, rh, _ := testDeps(t)
	db := pgmocks.NewMockDB(t)
	db.EXPECT().Ping(mock.Anything).Return(errors.New("connection refused"))
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready (db down): got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestRoadmapRecommendationsSuccess(t *testing.T) {
	fh, rh, db := testDeps(t)
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

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
	fh, rh, db := testDeps(t)
	router := NewRouter(fh, rh, NewHealthHandler(db), testConfig(), zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roadmap/recommendations", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing header: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}
