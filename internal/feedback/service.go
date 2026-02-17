package feedback

import (
	"context"
	"encoding/json"
	"time"
	"unicode"

	"github.com/MathTrail/mentor-api/internal/clients"
	"github.com/MathTrail/mentor-api/internal/strategy"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// Service defines the interface for feedback processing
type Service interface {
	ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error)
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	repo          Repository
	profileClient clients.ProfileClient
	analyzer      *strategy.Analyzer
	logger        *zap.Logger
}

// NewService creates a new feedback service
func NewService(
	repo Repository,
	profileClient clients.ProfileClient,
	analyzer *strategy.Analyzer,
	logger *zap.Logger,
) Service {
	return &serviceImpl{
		repo:          repo,
		profileClient: profileClient,
		analyzer:      analyzer,
		logger:        logger,
	}
}

// ProcessFeedback processes student feedback and stores it in the database
// Debezium CDC will automatically publish events to Kafka by monitoring the feedback table
func (s *serviceImpl) ProcessFeedback(ctx context.Context, req *FeedbackRequest) (*StrategyUpdate, error) {
	// 1. Detect language if not provided
	language := req.Language
	if language == "" {
		language = detectLanguage(req.Message)
	}

	// 2. Rule-based keyword analysis
	difficulty, adjustment := s.analyzer.Analyze(req.Message, language)

	// 3. Build strategy snapshot (current state)
	strategySnapshot := map[string]interface{}{
		"difficulty_weight": 1.0 + adjustment,
		"timestamp":         time.Now().Unix(),
		"feedback_based":    true,
		"language":          language,
		"sentiment":         difficulty,
	}

	// 4. Save feedback to PostgreSQL
	snapshotJSON, err := json.Marshal(strategySnapshot)
	if err != nil {
		s.logger.Error("failed to marshal strategy snapshot", zap.Error(err))
		return nil, err
	}

	feedback := &Feedback{
		StudentID:           req.StudentID,
		Message:             req.Message,
		PerceivedDifficulty: difficulty,
		StrategySnapshot:    datatypes.JSON(snapshotJSON),
	}

	if err := s.repo.Save(ctx, feedback); err != nil {
		s.logger.Error("failed to save feedback", zap.Error(err))
		return nil, err
	}

	s.logger.Info("feedback saved successfully",
		zap.String("student_id", req.StudentID.String()),
		zap.String("difficulty", difficulty),
		zap.Float64("adjustment", adjustment),
	)

	// 5. Build StrategyUpdate response (for HTTP response only)
	// NOTE: Debezium CDC will publish the actual event to Kafka by monitoring the feedback table
	update := &StrategyUpdate{
		StudentID:            req.StudentID,
		TaskID:               req.TaskID,
		DifficultyAdjustment: adjustment,
		TopicWeights:         calculateTopicWeights(adjustment),
		Sentiment:            difficulty,
		StrategySnapshot:     strategySnapshot,
		Timestamp:            time.Now(),
	}

	return update, nil
}

// detectLanguage performs simple language detection based on Cyrillic characters
// Returns "ru" if Cyrillic characters are found, otherwise "en"
func detectLanguage(text string) string {
	for _, r := range text {
		if unicode.In(r, unicode.Cyrillic) {
			return "ru"
		}
	}
	return "en"
}

// calculateTopicWeights generates topic-specific weight adjustments based on difficulty
// This is a simplified v1 implementation - will be enhanced with LLM analysis in v2
func calculateTopicWeights(difficultyAdjustment float64) map[string]float64 {
	// For now, apply uniform adjustment across all topics
	// Future: analyze feedback text to identify specific topics (algebra, geometry, etc.)
	return map[string]float64{
		"general": 1.0 + difficultyAdjustment,
	}
}
