package kafka

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestRecordCarrier_GetFound(t *testing.T) {
	c := NewRecordCarrier([]kgo.RecordHeader{
		{Key: "traceparent", Value: []byte("00-abc-def-01")},
	})
	if got := c.Get("traceparent"); got != "00-abc-def-01" {
		t.Errorf("Get: got %q, want %q", got, "00-abc-def-01")
	}
}

func TestRecordCarrier_GetNotFound(t *testing.T) {
	c := NewRecordCarrier([]kgo.RecordHeader{
		{Key: "other", Value: []byte("v")},
	})
	if got := c.Get("missing"); got != "" {
		t.Errorf("Get missing key: got %q, want %q", got, "")
	}
}

func TestRecordCarrier_Set(t *testing.T) {
	c := NewRecordCarrier(nil)
	c.Set("traceparent", "00-xyz-01")
	if got := c.Get("traceparent"); got != "00-xyz-01" {
		t.Errorf("after Set: got %q, want %q", got, "00-xyz-01")
	}
}

func TestRecordCarrier_Keys(t *testing.T) {
	c := NewRecordCarrier([]kgo.RecordHeader{
		{Key: "a", Value: []byte("1")},
		{Key: "b", Value: []byte("2")},
		{Key: "c", Value: []byte("3")},
	})
	keys := c.Keys()
	want := []string{"a", "b", "c"}
	if len(keys) != len(want) {
		t.Fatalf("Keys len: got %d, want %d", len(keys), len(want))
	}
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("Keys[%d]: got %q, want %q", i, keys[i], k)
		}
	}
}

func TestRecordCarrier_EmptyHeaders(t *testing.T) {
	c := NewRecordCarrier(nil)
	if got := c.Get("any"); got != "" {
		t.Errorf("Get on empty carrier: got %q, want empty", got)
	}
	if keys := c.Keys(); len(keys) != 0 {
		t.Errorf("Keys on empty carrier: got %v, want empty", keys)
	}
}
