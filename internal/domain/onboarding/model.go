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
}

// StudentOnboardingReadyEvent is the domain event received from students.onboarding.ready.
type StudentOnboardingReadyEvent struct {
	EventID    string
	UserID     string
	Email      string
	FirstName  string
	LastName   string
	Role       string
	OccurredAt string
}
