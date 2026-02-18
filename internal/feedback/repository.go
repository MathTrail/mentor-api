package feedback

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for feedback data access
type Repository interface {
	Save(ctx context.Context, feedback *Feedback) error
	GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error)
}

// repositoryImpl implements the Repository interface using pgx
type repositoryImpl struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new feedback repository
func NewRepository(pool *pgxpool.Pool) Repository {
	return &repositoryImpl{pool: pool}
}

// Save inserts a new feedback record into the database
func (r *repositoryImpl) Save(ctx context.Context, f *Feedback) error {
	const query = `
		INSERT INTO feedback (student_id, message, perceived_difficulty, strategy_snapshot)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		f.StudentID, f.Message, f.PerceivedDifficulty, f.StrategySnapshot,
	).Scan(&f.ID, &f.CreatedAt)
}

// GetLatestByStudent retrieves the most recent feedback for a student
func (r *repositoryImpl) GetLatestByStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]Feedback, error) {
	const query = `
		SELECT id, student_id, message, perceived_difficulty, strategy_snapshot, created_at
		FROM feedback
		WHERE student_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedbacks []Feedback
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(
			&f.ID, &f.StudentID, &f.Message,
			&f.PerceivedDifficulty, &f.StrategySnapshot, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, f)
	}

	return feedbacks, rows.Err()
}
