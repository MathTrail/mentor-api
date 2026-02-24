package roadmap

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation is the response returned to the student with personalised
// learning focus areas. Currently populated by a stub; a real LLM-based
// implementation will replace the stub in a future iteration.
type Recommendation struct {
	StudentID   uuid.UUID `json:"student_id"`
	FocusAreas  []string  `json:"focus_areas"`
	Message     string    `json:"message"`
	GeneratedAt time.Time `json:"generated_at"`
}
