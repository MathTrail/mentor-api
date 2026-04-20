package feedback

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MathTrail/mentor-api/internal/clients"
	"go.uber.org/zap"
)

// Service defines the interface for feedback processing
type Service interface {
	ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error)
}

// service implements the Service interface
type service struct {
	repo    Repository
	client  clients.FeedbackClient
	timeout time.Duration
	logger  *zap.Logger
}

// NewService creates a new feedback service
func NewService(
	repo Repository,
	client clients.FeedbackClient,
	timeout time.Duration,
	logger *zap.Logger,
) Service {
	return &service{
		repo:    repo,
		client:  client,
		timeout: timeout,
		logger:  logger,
	}
}

// ProcessFeedback sends the student message to the LLM for analysis,
// persists the resulting strategy, and returns it to the caller.
func (s *service) ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error) {
	// 1. Analyse via LLM with a dedicated timeout to protect HTTP workers.
	// Use a child context so the LLM deadline does not bleed into the DB write.
	llmCtx, llmCancel := context.WithTimeout(ctx, s.timeout)
	defer llmCancel()

	result, err := s.client.AnalyzeFeedback(llmCtx, req.Message)
	if err != nil {
		s.logger.Error("LLM analysis failed",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
		)
		return nil, fmt.Errorf("analysis: %w", err)
	}

	if err := validateResult(result); err != nil {
		s.logger.Error("LLM returned invalid result",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
		)
		return nil, fmt.Errorf("invalid LLM result: %w", err)
	}

	// 2. Serialise the strategy snapshot for storage
	snapshotJSON, err := result.MarshalSnapshot()
	if err != nil {
		s.logger.Error("failed to marshal strategy snapshot", zap.Error(err))
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	// 3. Persist feedback + strategy to PostgreSQL.
	// If this fails the LLM tokens are already spent — log enough context
	// to diagnose the issue without re-running the analysis.
	feedback := &Feedback{
		StudentID:           req.StudentID,
		TaskID:              req.TaskID,
		Message:             req.Message,
		PerceivedDifficulty: result.PerceivedDifficulty,
		StrategySnapshot:    snapshotJSON,
	}

	if err := s.repo.Save(ctx, feedback); err != nil {
		s.logger.Error("failed to save feedback after LLM analysis",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
			zap.String("task_id", req.TaskID),
			zap.Any("llm_result", result),
		)
		return nil, err
	}

	s.logger.Info("feedback saved",
		zap.String("student_id", req.StudentID.String()),
		zap.String("perceived_difficulty", result.PerceivedDifficulty),
	)

	// 4. Build response using the DB-assigned timestamp for consistency.
	return &StrategyUpdate{
		StudentID:            req.StudentID,
		TaskID:               req.TaskID,
		DifficultyAdjustment: result.DifficultyAdjustment,
		TopicWeights:         result.TopicWeights,
		Sentiment:            result.Sentiment,
		StrategySnapshot:     result.StrategySnapshot,
		Timestamp:            feedback.CreatedAt,
	}, nil
}

// validateResult guards against malformed or empty LLM responses.
func validateResult(r *clients.StrategyResult) error {
	if r.PerceivedDifficulty == "" {
		return errors.New("perceived_difficulty is empty")
	}
	if r.Sentiment == "" {
		return errors.New("sentiment is empty")
	}
	if len(r.TopicWeights) == 0 {
		return errors.New("topic_weights is empty")
	}
	for topic, w := range r.TopicWeights {
		if w < 0 {
			return fmt.Errorf("topic_weights[%q] is negative: %v", topic, w)
		}
	}
	return nil
}
