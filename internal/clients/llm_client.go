package clients

import (
	"context"
	"encoding/json"
)

// StrategyResult holds the LLM-determined strategy for a student.
type StrategyResult struct {
	PerceivedDifficulty  string                 `json:"perceived_difficulty"`
	DifficultyAdjustment float64                `json:"difficulty_adjustment"`
	TopicWeights         map[string]float64     `json:"topic_weights"`
	Sentiment            string                 `json:"sentiment"`
	StrategySnapshot     map[string]interface{} `json:"strategy_snapshot"`
}

// LLMClient defines the interface for LLM-based feedback analysis.
// The implementation will call an external LLM (OpenAI / Claude / etc.).
type LLMClient interface {
	AnalyzeFeedback(ctx context.Context, message string) (*StrategyResult, error)
}

// DefaultLLMClient is a stub implementation that returns a neutral strategy.
// Replace the body of AnalyzeFeedback with a real LLM call when ready.
type DefaultLLMClient struct{}

// NewLLMClient creates a new LLM client.
func NewLLMClient() LLMClient {
	return &DefaultLLMClient{}
}

// AnalyzeFeedback returns a hard-coded neutral strategy.
// TODO: replace with a real LLM call.
func (c *DefaultLLMClient) AnalyzeFeedback(_ context.Context, _ string) (*StrategyResult, error) {
	return &StrategyResult{
		PerceivedDifficulty:  "ok",
		DifficultyAdjustment: 0.0,
		TopicWeights:         map[string]float64{"general": 1.0},
		Sentiment:            "neutral",
		StrategySnapshot: map[string]interface{}{
			"difficulty_weight": 1.0,
			"feedback_based":    true,
			"sentiment":         "neutral",
		},
	}, nil
}

// MarshalSnapshot serialises StrategySnapshot to JSON bytes.
func (r *StrategyResult) MarshalSnapshot() (json.RawMessage, error) {
	return json.Marshal(r.StrategySnapshot)
}
