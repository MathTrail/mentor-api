package strategy

import (
	"strings"
)

// Analyzer provides rule-based sentiment analysis using keyword matching
type Analyzer struct {
	hardKeywords map[string][]string
	easyKeywords map[string][]string
}

// NewAnalyzer creates a new strategy analyzer with predefined keywords
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		hardKeywords: map[string][]string{
			"en": {"hard", "difficult", "challenging", "tough", "complex", "confusing", "can't solve", "too hard", "struggle"},
		},
		easyKeywords: map[string][]string{
			"en": {"easy", "simple", "too easy", "boring", "trivial", "very easy", "super easy"},
		},
	}
}

// Analyze performs keyword-based sentiment analysis on feedback text
// Returns difficulty level ("hard", "easy", "ok") and difficulty adjustment (-0.15 to +0.15)
func (a *Analyzer) Analyze(text, language string) (difficulty string, adjustment float64) {
	textLower := strings.ToLower(text)

	// Check for "hard" keywords first
	for _, keyword := range a.hardKeywords[language] {
		if strings.Contains(textLower, keyword) {
			return "hard", -0.15 // Decrease difficulty by 15%
		}
	}

	// Check for "easy" keywords
	for _, keyword := range a.easyKeywords[language] {
		if strings.Contains(textLower, keyword) {
			return "easy", 0.15 // Increase difficulty by 15%
		}
	}

	// No keywords matched - neutral feedback
	return "ok", 0.0
}
