package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/MathTrail/mentor-api/internal/database"
)

// Repository defines the interface for feedback data access
type Repository interface {
	Save(ctx context.Context, feedback *Feedback) error
	GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error)
}

// repositoryImpl implements the Repository interface using a Dapr PostgreSQL binding
type repositoryImpl struct {
	db *database.DaprDB
}

// NewRepository creates a new feedback repository backed by the given Dapr binding.
func NewRepository(db *database.DaprDB) Repository {
	return &repositoryImpl{db: db}
}

// Save inserts a new feedback record and populates ID and CreatedAt from the RETURNING clause.
func (r *repositoryImpl) Save(ctx context.Context, f *Feedback) error {
	const query = `
		INSERT INTO feedback (student_id, message, perceived_difficulty, strategy_snapshot)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	strategyJSON, err := json.Marshal(f.StrategySnapshot)
	if err != nil {
		return fmt.Errorf("feedback: marshal strategy_snapshot: %w", err)
	}

	rows, err := r.db.Query(ctx, query,
		f.StudentID.String(),
		f.Message,
		f.PerceivedDifficulty,
		string(strategyJSON),
	)
	if err != nil {
		return fmt.Errorf("feedback: save: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("feedback: save returned no rows")
	}

	if err := scanFeedbackID(rows[0], f); err != nil {
		return fmt.Errorf("feedback: scan save result: %w", err)
	}
	return nil
}

// GetLatestByStudent retrieves the most recent feedback for a student.
func (r *repositoryImpl) GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error) {
	const query = `
		SELECT id, student_id, message, perceived_difficulty, strategy_snapshot, created_at
		FROM feedback
		WHERE student_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, studentID.String(), limit)
	if err != nil {
		return nil, fmt.Errorf("feedback: get latest: %w", err)
	}

	feedbacks := make([]Feedback, 0, len(rows))
	for _, row := range rows {
		f, err := scanFeedback(row)
		if err != nil {
			return nil, fmt.Errorf("feedback: scan row: %w", err)
		}
		feedbacks = append(feedbacks, f)
	}
	return feedbacks, nil
}

// scanFeedbackID reads id and created_at from a Save RETURNING row.
func scanFeedbackID(row map[string]any, f *Feedback) error {
	idStr, ok := row["id"].(string)
	if !ok {
		return fmt.Errorf("id is not a string: %v", row["id"])
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse id: %w", err)
	}
	f.ID = id

	createdAt, err := parseTime(row["created_at"])
	if err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}
	f.CreatedAt = createdAt
	return nil
}

// scanFeedback deserializes a full feedback row from the Dapr binding JSON response.
func scanFeedback(row map[string]any) (Feedback, error) {
	var f Feedback

	idStr, ok := row["id"].(string)
	if !ok {
		return f, fmt.Errorf("id is not a string: %v", row["id"])
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return f, fmt.Errorf("parse id: %w", err)
	}
	f.ID = id

	sidStr, ok := row["student_id"].(string)
	if !ok {
		return f, fmt.Errorf("student_id is not a string: %v", row["student_id"])
	}
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		return f, fmt.Errorf("parse student_id: %w", err)
	}
	f.StudentID = sid

	f.Message, _ = row["message"].(string)
	f.PerceivedDifficulty, _ = row["perceived_difficulty"].(string)

	// strategy_snapshot is a JSONB column; Dapr returns it as a map. Re-marshal to json.RawMessage.
	snapshotBytes, err := json.Marshal(row["strategy_snapshot"])
	if err != nil {
		return f, fmt.Errorf("marshal strategy_snapshot: %w", err)
	}
	f.StrategySnapshot = json.RawMessage(snapshotBytes)

	createdAt, err := parseTime(row["created_at"])
	if err != nil {
		return f, fmt.Errorf("parse created_at: %w", err)
	}
	f.CreatedAt = createdAt

	return f, nil
}

// parseTime converts the Dapr binding's timestamp representation to time.Time.
// PostgreSQL timestamps arrive as RFC3339 strings.
func parseTime(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected string, got %T", v)
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
