package clients

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewProfileClientNotNil(t *testing.T) {
	c := NewProfileClient()
	if c == nil {
		t.Error("NewProfileClient returned nil")
	}
}

func TestGetProfileEchoesStudentID(t *testing.T) {
	c := NewProfileClient()
	id := uuid.New()
	profile, err := c.GetProfile(context.Background(), id)
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if profile.StudentID != id {
		t.Errorf("StudentID: got %v, want %v", profile.StudentID, id)
	}
}

func TestGetProfileHasExpectedSkills(t *testing.T) {
	c := NewProfileClient()
	profile, err := c.GetProfile(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	for _, skill := range []string{"algebra", "geometry", "logic"} {
		if _, ok := profile.Skills[skill]; !ok {
			t.Errorf("expected skill %q in profile", skill)
		}
	}
}
