package clients

import (
	"context"
)

// LLMClient defines the interface for LLM-based feedback analysis
// This will be implemented in v2 with OpenAI/Claude API integration
type LLMClient interface {
	AnalyzeFeedback(ctx context.Context, text string) (string, error)
}

// MockLLMClient is a mock implementation for development
type MockLLMClient struct{}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() LLMClient {
	return &MockLLMClient{}
}

// AnalyzeFeedback returns mock sentiment analysis
// Future: Replace with actual OpenAI/Claude API calls
func (m *MockLLMClient) AnalyzeFeedback(ctx context.Context, text string) (string, error) {
	return "neutral", nil
}
