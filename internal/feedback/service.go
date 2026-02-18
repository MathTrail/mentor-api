package feedback

import (
	"context"
	"time"

	"github.com/MathTrail/mentor-api/internal/clients"
	"go.uber.org/zap"
)

// Service defines the interface for feedback processing
type Service interface {
	ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error)
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	repo      Repository
	llmClient clients.LLMClient
	logger    *zap.Logger
}

// NewService creates a new feedback service
func NewService(
	repo Repository,
	llmClient clients.LLMClient,
	logger *zap.Logger,
) Service {
	return &serviceImpl{
		repo:      repo,
		llmClient: llmClient,
		logger:    logger,
	}
}

// ProcessFeedback sends the student message to the LLM for analysis,
// persists the resulting strategy, and returns it to the caller.
func (s *serviceImpl) ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error) {
	// 1. Delegate analysis to LLM (currently a mock)
	result, err := s.llmClient.AnalyzeFeedback(ctx, req.Message)
	if err != nil {
		s.logger.Error("LLM analysis failed", zap.Error(err))
		return nil, err
	}

	// 2. Serialise the strategy snapshot for storage
	snapshotJSON, err := result.MarshalSnapshot()
	if err != nil {
		s.logger.Error("failed to marshal strategy snapshot", zap.Error(err))
		return nil, err
	}

	// 3. Persist feedback + strategy to PostgreSQL
	feedback := &Feedback{
		StudentID:           req.StudentID,
		Message:             req.Message,
		PerceivedDifficulty: result.PerceivedDifficulty,
		StrategySnapshot:    snapshotJSON,
	}

	if err := s.repo.Save(ctx, feedback); err != nil {
		s.logger.Error("failed to save feedback", zap.Error(err))
		return nil, err
	}

	s.logger.Info("feedback saved",
		zap.String("student_id", req.StudentID.String()),
		zap.String("perceived_difficulty", result.PerceivedDifficulty),
	)

	// 4. Build response
	return &StrategyUpdate{
		StudentID:            req.StudentID,
		TaskID:               req.TaskID,
		DifficultyAdjustment: result.DifficultyAdjustment,
		TopicWeights:         result.TopicWeights,
		Sentiment:            result.Sentiment,
		StrategySnapshot:     result.StrategySnapshot,
		Timestamp:            time.Now(),
	}, nil
}
