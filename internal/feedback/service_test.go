package feedback

import (
	"context"
	"errors"
	"testing"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// mockRepository is a test double for feedback.Repository.
type mockRepository struct {
	saveFn             func(ctx context.Context, f *Feedback) error
	getLatestByStudent func(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error)
}

func (m *mockRepository) Save(ctx context.Context, f *Feedback) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, f)
	}
	f.ID = uuid.New()
	return nil
}

func (m *mockRepository) GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error) {
	if m.getLatestByStudent != nil {
		return m.getLatestByStudent(ctx, studentID, limit)
	}
	return nil, nil
}

func TestProcessFeedback_Success(t *testing.T) {
	repo := &mockRepository{}
	llm := clients.NewLLMClient()
	logger := zap.NewNop()

	svc := NewService(repo, llm, logger)

	req := &FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "This was a bit hard",
	}

	update, err := svc.ProcessFeedback(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if update.StudentID != req.StudentID {
		t.Errorf("student_id mismatch: got %v, want %v", update.StudentID, req.StudentID)
	}
	if update.TaskID != req.TaskID {
		t.Errorf("task_id mismatch: got %v, want %v", update.TaskID, req.TaskID)
	}
	if update.Sentiment != "neutral" {
		t.Errorf("sentiment: got %q, want %q", update.Sentiment, "neutral")
	}
}

func TestProcessFeedback_RepoError(t *testing.T) {
	repoErr := errors.New("connection refused")
	repo := &mockRepository{
		saveFn: func(_ context.Context, _ *Feedback) error { return repoErr },
	}
	llm := clients.NewLLMClient()
	logger := zap.NewNop()

	svc := NewService(repo, llm, logger)

	_, err := svc.ProcessFeedback(context.Background(), &FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-2",
		Message:   "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error mismatch: got %v, want %v", err, repoErr)
	}
}
