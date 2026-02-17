package feedback

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for feedback data access
type Repository interface {
	Save(ctx context.Context, feedback *Feedback) error
	GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error)
}

// repositoryImpl implements the Repository interface using GORM
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new feedback repository
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Save inserts a new feedback record into the database
func (r *repositoryImpl) Save(ctx context.Context, f *Feedback) error {
	return r.db.WithContext(ctx).Create(f).Error
}

// GetLatestByStudent retrieves the most recent feedback for a student
func (r *repositoryImpl) GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error) {
	var feedbacks []Feedback
	err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("created_at DESC").
		Limit(limit).
		Find(&feedbacks).Error
	return feedbacks, err
}
