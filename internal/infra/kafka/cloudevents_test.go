package kafka

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

func headers(pairs ...string) []kgo.RecordHeader {
	h := make([]kgo.RecordHeader, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		h = append(h, kgo.RecordHeader{Key: pairs[i], Value: []byte(pairs[i+1])})
	}
	return h
}

func TestExtractCloudEventHeaders_AllFields(t *testing.T) {
	h := headers(
		"ce-specversion", "1.0",
		"ce-type", "com.mathtrail.students.onboarding.ready",
		"ce-source", "/students",
		"ce-id", "abc-123",
		"ce-time", "2024-01-15T10:30:00Z",
	)
	ce := ExtractCloudEventHeaders(h)

	if ce.SpecVersion != "1.0" {
		t.Errorf("SpecVersion: got %q, want %q", ce.SpecVersion, "1.0")
	}
	if ce.Type != "com.mathtrail.students.onboarding.ready" {
		t.Errorf("Type: got %q", ce.Type)
	}
	if ce.Source != "/students" {
		t.Errorf("Source: got %q", ce.Source)
	}
	if ce.ID != "abc-123" {
		t.Errorf("ID: got %q", ce.ID)
	}
	if ce.Time != "2024-01-15T10:30:00Z" {
		t.Errorf("Time: got %q", ce.Time)
	}
}

func TestExtractCloudEventHeaders_Partial(t *testing.T) {
	h := headers("ce-id", "xyz", "ce-type", "some.type")
	ce := ExtractCloudEventHeaders(h)

	if ce.ID != "xyz" {
		t.Errorf("ID: got %q, want %q", ce.ID, "xyz")
	}
	if ce.Type != "some.type" {
		t.Errorf("Type: got %q, want %q", ce.Type, "some.type")
	}
	if ce.SpecVersion != "" || ce.Source != "" || ce.Time != "" {
		t.Errorf("unexpected non-empty fields: specversion=%q source=%q time=%q",
			ce.SpecVersion, ce.Source, ce.Time)
	}
}

func TestExtractCloudEventHeaders_Empty(t *testing.T) {
	ce := ExtractCloudEventHeaders(nil)

	if ce.SpecVersion != "" || ce.Type != "" || ce.Source != "" || ce.ID != "" || ce.Time != "" {
		t.Error("expected all fields to be empty for nil headers")
	}
}

func TestExtractCloudEventHeaders_UnknownKeys(t *testing.T) {
	h := headers(
		"traceparent", "00-abc-def-01",
		"x-custom", "value",
		"ce-id", "known",
	)
	ce := ExtractCloudEventHeaders(h)

	if ce.ID != "known" {
		t.Errorf("ID: got %q, want %q", ce.ID, "known")
	}
	// Unknown keys must not bleed into any CE field.
	if ce.SpecVersion != "" || ce.Type != "" || ce.Source != "" || ce.Time != "" {
		t.Errorf("unknown keys leaked into CE fields: %+v", ce)
	}
}
