package feedback

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// mockService is a test double for feedback.Service.
type mockService struct {
	processFn func(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error)
}

func (m *mockService) ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error) {
	if m.processFn != nil {
		return m.processFn(ctx, req)
	}
	return &StrategyUpdate{
		StudentID: req.StudentID,
		TaskID:    req.TaskID,
		Timestamp: time.Now(),
	}, nil
}

func testRouter(ctrl *Controller) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/feedback", ctrl.SubmitFeedback)
	return r
}

func TestSubmitFeedback_Success(t *testing.T) {
	svc := &mockService{}
	ctrl := NewController(svc, zap.NewNop())
	router := testRouter(ctrl)

	body, _ := json.Marshal(FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "This was easy",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var update StrategyUpdate
	if err := json.Unmarshal(w.Body.Bytes(), &update); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
}

func TestSubmitFeedback_InvalidJSON(t *testing.T) {
	svc := &mockService{}
	ctrl := NewController(svc, zap.NewNop())
	router := testRouter(ctrl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSubmitFeedback_MissingFields(t *testing.T) {
	svc := &mockService{}
	ctrl := NewController(svc, zap.NewNop())
	router := testRouter(ctrl)

	body, _ := json.Marshal(map[string]string{"message": "hello"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSubmitFeedback_ServiceError(t *testing.T) {
	svc := &mockService{
		processFn: func(_ context.Context, _ *FeedbackRequest) (*StrategyUpdate, error) {
			return nil, errors.New("llm timeout")
		},
	}
	ctrl := NewController(svc, zap.NewNop())
	router := testRouter(ctrl)

	body, _ := json.Marshal(FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
