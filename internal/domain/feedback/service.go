package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/MathTrail/mentor-api/internal/clients"
	"go.uber.org/zap"
)

// FeedbackService defines the interface for feedback processing
// NOSONAR: interface will be extended with additional methods
type FeedbackService interface {
	ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error)
}

// serviceImpl implements the FeedbackService interface
type serviceImpl struct {
	repo       Repository
	llmClient  clients.FeedbackClient
	llmTimeout time.Duration
	logger     *zap.Logger
}

// NewService creates a new feedback service
func NewService(
	repo Repository,
	llmClient clients.FeedbackClient,
	llmTimeout time.Duration,
	logger *zap.Logger,
) FeedbackService {
	return &serviceImpl{
		repo:       repo,
		llmClient:  llmClient,
		llmTimeout: llmTimeout,
		logger:     logger,
	}
}

// ProcessFeedback sends the student message to the LLM for analysis,
// persists the resulting strategy, and returns it to the caller.
func (s *serviceImpl) ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error) {
	// 1. Analyse via LLM with a dedicated timeout to protect HTTP workers.
	llmCtx, cancel := context.WithTimeout(ctx, s.llmTimeout)
	defer cancel()

	result, err := s.llmClient.AnalyzeFeedback(llmCtx, req.Message)
	if err != nil {
		s.logger.Error("LLM analysis failed",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
		)
		return nil, fmt.Errorf("analysis: %w", err)
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
		Message:             req.Message,
		PerceivedDifficulty: result.PerceivedDifficulty,
		StrategySnapshot:    snapshotJSON,
	}

	if err := s.repo.Save(ctx, feedback); err != nil {
		s.logger.Error("failed to save feedback after LLM analysis",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
			zap.String("task_id", req.TaskID),
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
