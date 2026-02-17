package feedback

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Feedback represents student feedback with perceived difficulty and strategy snapshot
type Feedback struct {
	ID                  uuid.UUID       `json:"id"`
	StudentID           uuid.UUID       `json:"student_id"`
	Message             string          `json:"message"`
	PerceivedDifficulty string          `json:"perceived_difficulty"`
	StrategySnapshot    json.RawMessage `json:"strategy_snapshot"`
	CreatedAt           time.Time       `json:"created_at"`
}

// FeedbackRequest is the HTTP request DTO for submitting feedback
type FeedbackRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	TaskID    string    `json:"task_id" binding:"required"`
	Message   string    `json:"message" binding:"required,max=5000"`
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
