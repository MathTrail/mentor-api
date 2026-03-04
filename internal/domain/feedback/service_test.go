package feedback

import (
	"context"
	"errors"
	"testing"
	"time"

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
	f.CreatedAt = time.Now().UTC()
	return nil
}

func (m *mockRepository) GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error) {
	if m.getLatestByStudent != nil {
		return m.getLatestByStudent(ctx, studentID, limit)
	}
	return nil, nil
}

// mockFeedbackAnalyzer is a test double for clients.FeedbackAnalyzer that blocks until
// the context expires — used to verify the per-call LLM timeout.
type mockFeedbackAnalyzer struct {
	delay time.Duration
}

func (m *mockFeedbackAnalyzer) AnalyzeFeedback(ctx context.Context, _ string) (*clients.StrategyResult, error) {
	select {
	case <-time.After(m.delay):
		return &clients.StrategyResult{
			PerceivedDifficulty: "ok",
			Sentiment:           "neutral",
			StrategySnapshot:    map[string]interface{}{"feedback_based": true},
			TopicWeights:        map[string]float64{"general": 1.0},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestProcessFeedbackSuccess(t *testing.T) {
	repo := &mockRepository{}
	llm := clients.NewLLMClient()
	logger := zap.NewNop()

	svc := NewService(repo, llm, 10*time.Second, logger)

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
	if update.Timestamp.IsZero() {
		t.Error("Timestamp should be non-zero (populated from DB CreatedAt)")
	}
}

func TestProcessFeedbackRepoError(t *testing.T) {
	repoErr := errors.New("connection refused")
	repo := &mockRepository{
		saveFn: func(_ context.Context, _ *Feedback) error { return repoErr },
	}
	llm := clients.NewLLMClient()
	logger := zap.NewNop()

	svc := NewService(repo, llm, 10*time.Second, logger)

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

func TestProcessFeedbackLLMTimeout(t *testing.T) {
	repo := &mockRepository{}
	// Mock LLM that blocks for 2 seconds — well beyond the 50ms timeout.
	llm := &mockFeedbackAnalyzer{delay: 2 * time.Second}
	logger := zap.NewNop()

	svc := NewService(repo, llm, 50*time.Millisecond, logger)

	_, err := svc.ProcessFeedback(context.Background(), &FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-3",
		Message:   "slow model",
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}
