package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/database"
	"github.com/MathTrail/mentor-api/internal/feedback"
	"go.uber.org/zap"
)

// --- test doubles ---

// mockDB implements database.DB for health check testing.
type mockDB struct {
	pingErr error
}

func (m *mockDB) Query(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error { return nil }
func (m *mockDB) Ping(_ context.Context) error                     { return m.pingErr }

// Compile-time interface check.
var _ database.DB = (*mockDB)(nil)

// mockService implements feedback.Service for controller construction.
type mockService struct{}

func (m *mockService) ProcessFeedback(_ context.Context, req *feedback.FeedbackRequest) (*feedback.StrategyUpdate, error) {
	return &feedback.StrategyUpdate{StudentID: req.StudentID}, nil
}

// --- tests ---

func TestHealthStartup(t *testing.T) {
	ctrl := feedback.NewController(&mockService{}, zap.NewNop())
	router := NewRouter(ctrl, &mockDB{}, zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("startup: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthLiveness(t *testing.T) {
	ctrl := feedback.NewController(&mockService{}, zap.NewNop())
	router := NewRouter(ctrl, &mockDB{}, zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("liveness: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReady_OK(t *testing.T) {
	ctrl := feedback.NewController(&mockService{}, zap.NewNop())
	router := NewRouter(ctrl, &mockDB{}, zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthReady_DBDown(t *testing.T) {
	ctrl := feedback.NewController(&mockService{}, zap.NewNop())
	router := NewRouter(ctrl, &mockDB{pingErr: errors.New("connection refused")}, zap.NewNop())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready (db down): got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
