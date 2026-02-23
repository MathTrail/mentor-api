package roadmap

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestGetRecommendations_Stub(t *testing.T) {
	svc := NewService(zap.NewNop())

	rec, err := svc.GetRecommendations(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if rec == nil {
		t.Fatal("expected non-nil recommendation")
	}
	if len(rec.FocusAreas) == 0 {
		t.Error("expected at least one focus area")
	}
	if rec.Message == "" {
		t.Error("expected non-empty message")
	}
	if rec.GeneratedAt.IsZero() {
		t.Error("expected non-zero GeneratedAt")
	}
}

func TestGetRecommendations_StudentIDPreserved(t *testing.T) {
	svc := NewService(zap.NewNop())
	id := uuid.New()

	rec, err := svc.GetRecommendations(context.Background(), id)
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if rec.StudentID != id {
		t.Errorf("StudentID = %v, want %v", rec.StudentID, id)
	}
}
