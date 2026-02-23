package database_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/MathTrail/mentor-api/internal/database"
)

func TestDecodeRow(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	row := map[string]any{
		"id":         id.String(),
		"name":       "Alice",
		"created_at": now.Format(time.RFC3339Nano),
		"meta":       map[string]any{"key": "value"},
	}

	type target struct {
		ID        uuid.UUID       `json:"id"`
		Name      string          `json:"name"`
		CreatedAt time.Time       `json:"created_at"`
		Meta      json.RawMessage `json:"meta"`
	}

	got, err := database.DecodeRow[target](row)
	if err != nil {
		t.Fatalf("DecodeRow: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
	if got.Name != "Alice" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice")
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
	if string(got.Meta) != `{"key":"value"}` {
		t.Errorf("Meta = %s, want %s", got.Meta, `{"key":"value"}`)
	}
}

func TestDecodeRows(t *testing.T) {
	rows := []map[string]any{
		{"id": uuid.New().String(), "name": "Alice"},
		{"id": uuid.New().String(), "name": "Bob"},
	}

	type target struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}

	got, err := database.DecodeRows[target](rows)
	if err != nil {
		t.Fatalf("DecodeRows: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Alice" {
		t.Errorf("[0].Name = %q, want %q", got[0].Name, "Alice")
	}
	if got[1].Name != "Bob" {
		t.Errorf("[1].Name = %q, want %q", got[1].Name, "Bob")
	}
}

func TestDecodeRowsEmpty(t *testing.T) {
	type target struct {
		ID uuid.UUID `json:"id"`
	}

	got, err := database.DecodeRows[target]([]map[string]any{})
	if err != nil {
		t.Fatalf("DecodeRows: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil slice for empty input")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
