package clients

import (
	"context"

	"github.com/google/uuid"
)

// Profile represents student profile data
type Profile struct {
	StudentID uuid.UUID       `json:"student_id"`
	Name      string          `json:"name"`
	Skills    map[string]int  `json:"skills"`
}

// ProfileClient defines the interface for accessing student profile data
type ProfileClient interface {
	GetProfile(ctx context.Context, studentID uuid.UUID) (*Profile, error)
}

// MockProfileClient is a mock implementation for development
type MockProfileClient struct{}

// NewMockProfileClient creates a new mock profile client
func NewMockProfileClient() ProfileClient {
	return &MockProfileClient{}
}

// GetProfile returns mock profile data
func (m *MockProfileClient) GetProfile(ctx context.Context, studentID uuid.UUID) (*Profile, error) {
	return &Profile{
		StudentID: studentID,
		Name:      "Mock Student",
		Skills: map[string]int{
			"algebra":  5,
			"geometry": 3,
			"logic":    4,
		},
	}, nil
}
