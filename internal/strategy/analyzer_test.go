package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzer_English_Hard(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name     string
		message  string
		expected string
		adjExpected float64
	}{
		{"hard keyword", "this is too hard", "hard", -0.15},
		{"difficult keyword", "very difficult problem", "hard", -0.15},
		{"challenging keyword", "this is challenging", "hard", -0.15},
		{"can't solve", "I can't solve this", "hard", -0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty, adj := a.Analyze(tt.message, "en")
			assert.Equal(t, tt.expected, difficulty)
			assert.Equal(t, tt.adjExpected, adj)
		})
	}
}

func TestAnalyzer_English_Easy(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name     string
		message  string
		expected string
		adjExpected float64
	}{
		{"easy keyword", "this is easy", "easy", 0.15},
		{"too easy keyword", "too easy for me", "easy", 0.15},
		{"simple keyword", "very simple", "easy", 0.15},
		{"boring keyword", "it's boring", "easy", 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty, adj := a.Analyze(tt.message, "en")
			assert.Equal(t, tt.expected, difficulty)
			assert.Equal(t, tt.adjExpected, adj)
		})
	}
}

func TestAnalyzer_Russian_Hard(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name     string
		message  string
		expected string
		adjExpected float64
	}{
		{"сложно", "это слишком сложно", "hard", -0.15},
		{"трудно", "очень трудно", "hard", -0.15},
		{"не понимаю", "я не понимаю", "hard", -0.15},
		{"непонятно", "мне непонятно", "hard", -0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty, adj := a.Analyze(tt.message, "ru")
			assert.Equal(t, tt.expected, difficulty)
			assert.Equal(t, tt.adjExpected, adj)
		})
	}
}

func TestAnalyzer_Russian_Easy(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name     string
		message  string
		expected string
		adjExpected float64
	}{
		{"легко", "это легко", "easy", 0.15},
		{"просто", "совсем просто", "easy", 0.15},
		{"слишком легко", "слишком легко", "easy", 0.15},
		{"скучно", "скучно", "easy", 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty, adj := a.Analyze(tt.message, "ru")
			assert.Equal(t, tt.expected, difficulty)
			assert.Equal(t, tt.adjExpected, adj)
		})
	}
}

func TestAnalyzer_Neutral(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name    string
		message string
		lang    string
	}{
		{"English neutral", "I completed the task", "en"},
		{"Russian neutral", "Я решил задачу", "ru"},
		{"No keywords", "Thank you", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty, adj := a.Analyze(tt.message, tt.lang)
			assert.Equal(t, "ok", difficulty)
			assert.Equal(t, 0.0, adj)
		})
	}
}

func TestAnalyzer_CaseInsensitive(t *testing.T) {
	a := NewAnalyzer()

	difficulty1, adj1 := a.Analyze("THIS IS HARD", "en")
	difficulty2, adj2 := a.Analyze("this is hard", "en")

	assert.Equal(t, difficulty1, difficulty2)
	assert.Equal(t, adj1, adj2)
	assert.Equal(t, "hard", difficulty1)
	assert.Equal(t, -0.15, adj1)
}
