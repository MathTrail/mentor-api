package middleware

import "testing"

func TestSanitizeQuery_MasksSensitiveKeys(t *testing.T) {
	raw := "token=secret123&page=1&api_key=abc"
	got := sanitizeQuery(raw)

	// Verify sensitive keys are masked.
	for _, key := range []string{"token", "api_key"} {
		if !contains(got, key+"=%2A%2A%2A") && !contains(got, key+"=***") {
			t.Errorf("expected %s to be masked in %q", key, got)
		}
	}
	// Verify non-sensitive key is preserved.
	if !contains(got, "page=1") {
		t.Errorf("expected page=1 in %q", got)
	}
}

func TestSanitizeQuery_Empty(t *testing.T) {
	if got := sanitizeQuery(""); got != "" {
		t.Errorf("sanitizeQuery(\"\") = %q, want \"\"", got)
	}
}

func TestSanitizeQuery_NoSensitiveKeys(t *testing.T) {
	raw := "page=1&limit=10"
	got := sanitizeQuery(raw)
	if !contains(got, "page=1") || !contains(got, "limit=10") {
		t.Errorf("expected all params preserved in %q", got)
	}
}

func TestSanitizeQuery_CaseInsensitive(t *testing.T) {
	raw := "TOKEN=secret&Password=hunter2"
	got := sanitizeQuery(raw)
	for _, key := range []string{"TOKEN", "Password"} {
		if !contains(got, key+"=%2A%2A%2A") && !contains(got, key+"=***") {
			t.Errorf("expected %s to be masked in %q", key, got)
		}
	}
}

func TestIsInternalPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/health/ready", true},
		{"/health/liveness", true},
		{"/dapr/config", true},
		{"/metrics", true},
		{"/api/v1/feedback", false},
		{"/swagger/index.html", false},
	}
	for _, tt := range tests {
		if got := isInternalPath(tt.path); got != tt.want {
			t.Errorf("isInternalPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// contains checks if substr is in s (simple helper to avoid importing strings in tests).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
