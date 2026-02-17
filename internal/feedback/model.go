package feedback

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Feedback represents student feedback with perceived difficulty and strategy snapshot
type Feedback struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID           uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Message             string         `gorm:"type:text" json:"message"`
	PerceivedDifficulty string         `gorm:"type:difficulty_level;not null" json:"perceived_difficulty"`
	StrategySnapshot    datatypes.JSON `gorm:"type:jsonb;not null" json:"strategy_snapshot"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

// FeedbackRequest is the HTTP request DTO for submitting feedback
type FeedbackRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	TaskID    string    `json:"task_id" binding:"required"`
	Message   string    `json:"message" binding:"required,max=5000"`
	Language  string    `json:"language" binding:"omitempty,oneof=en ru"`
}

// StrategyUpdate is the response DTO returned after processing feedback
type StrategyUpdate struct {
	StudentID            uuid.UUID              `json:"student_id"`
	TaskID               string                 `json:"task_id"`
	DifficultyAdjustment float64                `json:"difficulty_adjustment"`
	TopicWeights         map[string]float64     `json:"topic_weights"`
	Sentiment            string                 `json:"sentiment"`
	StrategySnapshot     map[string]interface{} `json:"strategy_snapshot"`
	Timestamp            time.Time              `json:"timestamp"`
}

// ErrorResponse represents an HTTP error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
