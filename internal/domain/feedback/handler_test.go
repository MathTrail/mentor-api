package feedback_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	feedbackmocks "github.com/MathTrail/mentor-api/internal/domain/feedback/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	feedbackPath      = "/api/v1/feedback"
	contentTypeJSON   = "application/json"
	contentTypeHeader = "Content-Type"
	statusCodeFmt     = "status code: got %d, want %d"
)

func testFeedbackRouter(h *feedback.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST(feedbackPath, h.SubmitFeedback)
	return r
}

func TestSubmitFeedbackSuccess(t *testing.T) {
	svc := feedbackmocks.NewMockService(t)
	svc.EXPECT().ProcessFeedback(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *feedback.FeedbackRequest) (*feedback.StrategyUpdate, error) {
			return &feedback.StrategyUpdate{
				StudentID: req.StudentID,
				TaskID:    req.TaskID,
			}, nil
		})

	hdl := feedback.NewHandler(svc, zap.NewNop())
	router := testFeedbackRouter(hdl)

	body, _ := json.Marshal(feedback.FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "This was easy",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, feedbackPath, bytes.NewReader(body))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf(statusCodeFmt, w.Code, http.StatusOK)
	}

	var update feedback.StrategyUpdate
	if err := json.Unmarshal(w.Body.Bytes(), &update); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
}

func TestSubmitFeedbackInvalidJSON(t *testing.T) {
	svc := feedbackmocks.NewMockService(t)
	hdl := feedback.NewHandler(svc, zap.NewNop())
	router := testFeedbackRouter(hdl)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, feedbackPath, bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf(statusCodeFmt, w.Code, http.StatusBadRequest)
	}
}

func TestSubmitFeedbackMissingFields(t *testing.T) {
	svc := feedbackmocks.NewMockService(t)
	hdl := feedback.NewHandler(svc, zap.NewNop())
	router := testFeedbackRouter(hdl)

	body, _ := json.Marshal(map[string]string{"message": "hello"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, feedbackPath, bytes.NewReader(body))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf(statusCodeFmt, w.Code, http.StatusBadRequest)
	}
}

func TestSubmitFeedbackServiceError(t *testing.T) {
	svc := feedbackmocks.NewMockService(t)
	svc.EXPECT().ProcessFeedback(mock.Anything, mock.Anything).
		Return(nil, errors.New("llm timeout"))

	hdl := feedback.NewHandler(svc, zap.NewNop())
	router := testFeedbackRouter(hdl)

	body, _ := json.Marshal(feedback.FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, feedbackPath, bytes.NewReader(body))
	req.Header.Set(contentTypeHeader, contentTypeJSON)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf(statusCodeFmt, w.Code, http.StatusInternalServerError)
	}
}
