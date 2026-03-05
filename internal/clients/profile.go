package clients

import (
	"context"

	"github.com/google/uuid"
)

// Profile represents student profile data
type Profile struct {
	StudentID uuid.UUID      `json:"student_id"`
	Name      string         `json:"name"`
	Skills    map[string]int `json:"skills"`
}

// ProfileClient defines the interface for accessing student profile data
type ProfileClient interface { // NOSONAR: interface will be extended with additional methods
	GetProfile(ctx context.Context, studentID uuid.UUID) (*Profile, error)
}

type profileClient struct{}

// NewProfileClient creates a new profile client.
func NewProfileClient() ProfileClient {
	return &profileClient{}
}

// GetProfile returns a student profile.
func (c *profileClient) GetProfile(ctx context.Context, studentID uuid.UUID) (*Profile, error) {
	return &Profile{
		StudentID: studentID,
		Name:      "Student",
		Skills: map[string]int{
			"algebra":  5,
			"geometry": 3,
			"logic":    4,
		},
	}, nil
}
