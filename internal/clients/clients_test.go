package clients

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// --- LLM client tests ---

func TestNewLLMClient_NotNil(t *testing.T) {
	c := NewLLMClient()
	if c == nil {
		t.Error("NewLLMClient returned nil")
	}
}

func TestAnalyzeFeedback_StubValues(t *testing.T) {
	c := NewLLMClient()
	result, err := c.AnalyzeFeedback(context.Background(), "this was hard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sentiment != "neutral" {
		t.Errorf("Sentiment: got %q, want %q", result.Sentiment, "neutral")
	}
	if result.PerceivedDifficulty != "ok" {
		t.Errorf("PerceivedDifficulty: got %q, want %q", result.PerceivedDifficulty, "ok")
	}
	if result.DifficultyAdjustment != 0.0 {
		t.Errorf("DifficultyAdjustment: got %v, want 0.0", result.DifficultyAdjustment)
	}
	if result.TopicWeights["general"] != 1.0 {
		t.Errorf("TopicWeights[general]: got %v, want 1.0", result.TopicWeights["general"])
	}
}

func TestMarshalSnapshot_NonNil(t *testing.T) {
	result := &StrategyResult{
		StrategySnapshot: map[string]interface{}{
			"difficulty_weight": 1.0,
			"feedback_based":    true,
		},
	}
	raw, err := result.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot error: %v", err)
	}
	if !json.Valid(raw) {
		t.Errorf("MarshalSnapshot returned invalid JSON: %s", raw)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if _, ok := m["difficulty_weight"]; !ok {
		t.Error("expected key 'difficulty_weight' in snapshot JSON")
	}
}

func TestMarshalSnapshot_NilMap(t *testing.T) {
	result := &StrategyResult{StrategySnapshot: nil}
	raw, err := result.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot error: %v", err)
	}
	// json.Marshal(nil map) produces "null"
	if string(raw) != "null" {
		t.Errorf("expected %q, got %q", "null", string(raw))
	}
}

// --- Profile client tests ---

func TestNewProfileClient_NotNil(t *testing.T) {
	c := NewProfileClient()
	if c == nil {
		t.Error("NewProfileClient returned nil")
	}
}

func TestGetProfile_EchoesStudentID(t *testing.T) {
	c := NewProfileClient()
	id := uuid.New()
	profile, err := c.GetProfile(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.StudentID != id {
		t.Errorf("StudentID: got %v, want %v", profile.StudentID, id)
	}
}

func TestGetProfile_HasExpectedSkills(t *testing.T) {
	c := NewProfileClient()
	profile, err := c.GetProfile(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, skill := range []string{"algebra", "geometry", "logic"} {
		if _, ok := profile.Skills[skill]; !ok {
			t.Errorf("expected skill %q in profile", skill)
		}
	}
}
