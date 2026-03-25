package onboarding

import (
	"context"
	"time"

	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	"github.com/google/uuid"
)

// Repository persists onboarding recommendations.
type Repository struct {
	db postgres.DB
}

func NewRepository(db postgres.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts a new recommendation or updates the existing one if the incoming
// occurred_at is strictly newer. This ensures idempotency: redelivered events
// with the same or older timestamp are ignored.
func (r *Repository) Upsert(ctx context.Context, studentID uuid.UUID, eventID string, occurredAt time.Time) error {
	const q = `
		INSERT INTO recommendations (student_id, event_id, occurred_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (student_id) DO UPDATE
			SET event_id    = EXCLUDED.event_id,
			    occurred_at = EXCLUDED.occurred_at,
			    updated_at  = NOW()
		WHERE EXCLUDED.occurred_at > recommendations.occurred_at
	`
	return r.db.Exec(ctx, q, studentID, eventID, occurredAt)
}
