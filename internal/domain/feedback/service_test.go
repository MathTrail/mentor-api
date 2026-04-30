package feedback_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MathTrail/mentor-api/internal/clients"
	clientmocks "github.com/MathTrail/mentor-api/internal/clients/mocks"
	"github.com/MathTrail/mentor-api/internal/domain/feedback"
	feedbackmocks "github.com/MathTrail/mentor-api/internal/domain/feedback/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	unexpectedErrFmt = "unexpected error: %v"
	expectedErrFmt   = "expected error, got nil"
)

func TestProcessFeedbackSuccess(t *testing.T) {
	repo := feedbackmocks.NewMockRepository(t)
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, f *feedback.Feedback) error {
			f.ID = uuid.New()
			f.CreatedAt = time.Now().UTC()
			return nil
		})

	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).
		Return(&clients.StrategyResult{
			PerceivedDifficulty: "ok",
			Sentiment:           "neutral",
			TopicWeights:        map[string]float64{"general": 1.0},
			StrategySnapshot:    map[string]any{"feedback_based": true},
		}, nil)

	svc := feedback.NewService(repo, llm, 10*time.Second, zap.NewNop())

	req := &feedback.FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-1",
		Message:   "This was a bit hard",
	}

	update, err := svc.ProcessFeedback(context.Background(), req)
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
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

	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).
		Return(&clients.StrategyResult{
			PerceivedDifficulty: "ok",
			Sentiment:           "neutral",
			TopicWeights:        map[string]float64{"general": 1.0},
		}, nil)

	repo := feedbackmocks.NewMockRepository(t)
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(repoErr)

	svc := feedback.NewService(repo, llm, 10*time.Second, zap.NewNop())

	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{
		StudentID: uuid.New(),
		TaskID:    "task-2",
		Message:   "test",
	})
	if err == nil {
		t.Fatal(expectedErrFmt)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error mismatch: got %v, want %v", err, repoErr)
	}
}

func TestProcessFeedbackLLMTimeout(t *testing.T) {
	repo := feedbackmocks.NewMockRepository(t)

	llm := clientmocks.NewMockFeedbackClient(t)
	// Blocks for 2 seconds — well beyond the 50 ms service timeout.
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ string) (*clients.StrategyResult, error) {
			select {
			case <-time.After(2 * time.Second):
				return &clients.StrategyResult{
					PerceivedDifficulty: "ok",
					Sentiment:           "neutral",
					TopicWeights:        map[string]float64{"general": 1.0},
				}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})

	svc := feedback.NewService(repo, llm, 50*time.Millisecond, zap.NewNop())

	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{
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

// goodResult returns a valid StrategyResult to use as a base in validation tests.
func goodResult() *clients.StrategyResult {
	return &clients.StrategyResult{
		PerceivedDifficulty: "hard",
		Sentiment:           "frustrated",
		TopicWeights:        map[string]float64{"algebra": 1.0},
	}
}

func TestProcessFeedback_EmptyPerceivedDifficulty(t *testing.T) {
	r := goodResult()
	r.PerceivedDifficulty = ""
	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).Return(r, nil)
	svc := feedback.NewService(feedbackmocks.NewMockRepository(t), llm, time.Second, zap.NewNop())
	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{StudentID: uuid.New(), TaskID: "t1", Message: "m"})
	if err == nil {
		t.Fatal(expectedErrFmt)
	}
}

func TestProcessFeedback_EmptySentiment(t *testing.T) {
	r := goodResult()
	r.Sentiment = ""
	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).Return(r, nil)
	svc := feedback.NewService(feedbackmocks.NewMockRepository(t), llm, time.Second, zap.NewNop())
	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{StudentID: uuid.New(), TaskID: "t1", Message: "m"})
	if err == nil {
		t.Fatal(expectedErrFmt)
	}
}

func TestProcessFeedback_EmptyTopicWeights(t *testing.T) {
	r := goodResult()
	r.TopicWeights = nil
	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).Return(r, nil)
	svc := feedback.NewService(feedbackmocks.NewMockRepository(t), llm, time.Second, zap.NewNop())
	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{StudentID: uuid.New(), TaskID: "t1", Message: "m"})
	if err == nil {
		t.Fatal(expectedErrFmt)
	}
}

func TestProcessFeedback_NegativeTopicWeight(t *testing.T) {
	r := goodResult()
	r.TopicWeights = map[string]float64{"algebra": -0.5}
	llm := clientmocks.NewMockFeedbackClient(t)
	llm.EXPECT().AnalyzeFeedback(mock.Anything, mock.Anything).Return(r, nil)
	svc := feedback.NewService(feedbackmocks.NewMockRepository(t), llm, time.Second, zap.NewNop())
	_, err := svc.ProcessFeedback(context.Background(), &feedback.FeedbackRequest{StudentID: uuid.New(), TaskID: "t1", Message: "m"})
	if err == nil {
		t.Fatal(expectedErrFmt)
	}
}
