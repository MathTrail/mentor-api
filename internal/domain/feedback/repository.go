package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/MathTrail/mentor-api/internal/infra/postgres"
)

// Repository defines the interface for feedback data access
type Repository interface {
	Save(ctx context.Context, feedback *Feedback) error
	GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error)
}

// repositoryImpl implements the Repository interface.
type repositoryImpl struct {
	db postgres.DB
}

// NewRepository creates a new feedback repository backed by the given database.
func NewRepository(db postgres.DB) Repository {
	return &repositoryImpl{db: db}
}

// saveResult captures the RETURNING columns from an INSERT.
type saveResult struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// Save inserts a new feedback record and populates ID and CreatedAt from the RETURNING clause.
func (r *repositoryImpl) Save(ctx context.Context, f *Feedback) error {
	const query = `
		INSERT INTO feedback (student_id, message, perceived_difficulty, strategy_snapshot)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	rows, err := r.db.Query(ctx, query,
		f.StudentID.String(),
		f.Message,
		f.PerceivedDifficulty,
		string(f.StrategySnapshot),
	)
	if err != nil {
		return fmt.Errorf("feedback: save: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("feedback: save returned no rows")
	}

	res, err := postgres.DecodeRow[saveResult](rows[0])
	if err != nil {
		return fmt.Errorf("feedback: scan save result: %w", err)
	}
	f.ID = res.ID
	f.CreatedAt = res.CreatedAt
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

	return postgres.DecodeRows[Feedback](rows)
}
