package kafka

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeWireFormat builds a Confluent Protobuf Wire Format message:
// 5-byte header (magic + schema ID) + 1-byte message index (0x00 = first message) + payload.
func makeWireFormat(schemaID uint32, payload []byte) []byte {
	buf := make([]byte, confluentHeaderSize+1+len(payload))
	buf[0] = confluentMagicByte
	binary.BigEndian.PutUint32(buf[1:confluentHeaderSize], schemaID)
	buf[confluentHeaderSize] = 0x00 // message index: count=0 (first/only message)
	copy(buf[confluentHeaderSize+1:], payload)
	return buf
}

// --- ValidateAndUnwrap ---

func TestValidateAndUnwrap_Success(t *testing.T) {
	v := &TopicValidator{ExpectedID: 42, ExpectedSubject: "test"}
	data := makeWireFormat(42, []byte("hello"))

	got, err := v.ValidateAndUnwrap(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("payload: got %q, want %q", got, "hello")
	}
}

func TestValidateAndUnwrap_TooShort(t *testing.T) {
	v := &TopicValidator{ExpectedID: 42}
	_, err := v.ValidateAndUnwrap([]byte{0x00, 0x01, 0x02, 0x03}) // 4 bytes, need 5
	if err == nil {
		t.Error("expected error for too-short data, got nil")
	}
}

func TestValidateAndUnwrap_WrongMagicByte(t *testing.T) {
	v := &TopicValidator{ExpectedID: 42}
	data := makeWireFormat(42, []byte("payload"))
	data[0] = 0x01 // corrupt magic byte
	_, err := v.ValidateAndUnwrap(data)
	if err == nil {
		t.Error("expected error for wrong magic byte, got nil")
	}
}

func TestValidateAndUnwrap_SchemaMismatch(t *testing.T) {
	v := &TopicValidator{ExpectedID: 42, ExpectedSubject: "my-subject"}
	data := makeWireFormat(99, []byte("payload")) // ID 99 ≠ expected 42
	_, err := v.ValidateAndUnwrap(data)
	if err == nil {
		t.Error("expected schema mismatch error, got nil")
	}
}

// --- NewValidator ---

func registryServer(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestNewValidator_Success(t *testing.T) {
	srv := registryServer(t, http.StatusOK, map[string]any{"id": 7})
	defer srv.Close()

	v, err := NewValidator(srv.URL, "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ExpectedID != 7 {
		t.Errorf("ExpectedID: got %d, want 7", v.ExpectedID)
	}
	if v.ExpectedSubject != "test-subject" {
		t.Errorf("ExpectedSubject: got %q, want %q", v.ExpectedSubject, "test-subject")
	}
}

func TestNewValidator_HTTPError(t *testing.T) {
	srv := registryServer(t, http.StatusNotFound, map[string]any{"message": "not found"})
	defer srv.Close()

	_, err := NewValidator(srv.URL, "missing-subject")
	if err == nil {
		t.Error("expected error for HTTP 404, got nil")
	}
}

func TestNewValidator_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{broken json"))
	}))
	defer srv.Close()

	_, err := NewValidator(srv.URL, "any-subject")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestNewValidator_ZeroID(t *testing.T) {
	srv := registryServer(t, http.StatusOK, map[string]any{"id": 0})
	defer srv.Close()

	_, err := NewValidator(srv.URL, "zero-subject")
	if err == nil {
		t.Error("expected error for schema id 0, got nil")
	}
}

// ── skipMessageIndexes unit tests ──────────────────────────────────────────

func TestSkipMessageIndexes_Empty(t *testing.T) {
	_, err := skipMessageIndexes([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestSkipMessageIndexes_SingleMessage(t *testing.T) {
	// count=0 encodes as a single 0x00 byte; this represents the first (only) message.
	n, err := skipMessageIndexes([]byte{0x00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1", n)
	}
}

func TestSkipMessageIndexes_MultipleMessageIndexes(t *testing.T) {
	// count=2 (0x02), followed by two varint message indexes (0x00, 0x02).
	data := []byte{0x02, 0x00, 0x02}
	n, err := skipMessageIndexes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("n = %d, want 3", n)
	}
}

func TestSkipMessageIndexes_OverflowCount(t *testing.T) {
	// Ten 0xFF bytes produce a varint overflow (value > 64 bits).
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	_, err := skipMessageIndexes(data)
	if err == nil {
		t.Error("expected error for overflowed varint, got nil")
	}
}

func TestSkipMessageIndexes_TruncatedIndex(t *testing.T) {
	// count=1 but no message-index byte follows.
	_, err := skipMessageIndexes([]byte{0x01})
	if err == nil {
		t.Error("expected error for truncated message index, got nil")
	}
}
