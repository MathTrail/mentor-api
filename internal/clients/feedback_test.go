package clients

import (
	"context"
	"encoding/json"
	"testing"
)

const unexpectedErrorFmt = "unexpected error: %v"

func TestNewFeedbackClientNotNil(t *testing.T) {
	c := NewFeedbackClient()
	if c == nil {
		t.Error("NewFeedbackClient returned nil")
	}
}

func TestAnalyzeFeedbackStubValues(t *testing.T) {
	c := NewFeedbackClient()
	result, err := c.AnalyzeFeedback(context.Background(), "this was hard")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
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

func TestMarshalSnapshotNonNil(t *testing.T) {
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

func TestMarshalSnapshotNilMap(t *testing.T) {
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
