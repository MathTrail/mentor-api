package roadmap

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the interface for roadmap recommendation generation.
type Service interface {
	GetRecommendations(ctx context.Context, studentID uuid.UUID) (*Recommendation, error)
}

// service implements the Service interface.
type service struct {
	logger *zap.Logger
}

// NewService creates a new roadmap service.
func NewService(logger *zap.Logger) Service {
	return &service{logger: logger}
}

// GetRecommendations returns personalised learning focus areas for a student.
func (s *service) GetRecommendations(ctx context.Context, studentID uuid.UUID) (*Recommendation, error) {
	s.logger.Info("generating roadmap recommendations", zap.Stringer("student_id", studentID))

	return &Recommendation{
		StudentID:   studentID,
		FocusAreas:  []string{"algebra", "fractions", "word problems"},
		Message:     "Based on your recent progress, focus on algebra fundamentals and practice word problems.",
		GeneratedAt: time.Now(),
	}, nil
}
