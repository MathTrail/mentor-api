package onboarding

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation is the persisted record created when a student completes onboarding.
type Recommendation struct {
	ID         uuid.UUID
	StudentID  uuid.UUID
	EventID    string
	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
