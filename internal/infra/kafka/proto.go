package kafka

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// confluentMagicByte is the first byte of every Confluent Wire Format message (always 0x00).
// confluentHeaderSize is the size of the Confluent Wire Format header: 1 magic byte + 4 schema ID bytes.
// schemaFetchMaxAttempts is the number of attempts to fetch the schema ID from Apicurio at startup.
const (
	confluentMagicByte     = 0x00
	confluentHeaderSize    = 5
	schemaFetchMaxAttempts = 5
)

// TopicValidator validates Confluent Wire Format messages against a pinned schema ID.
// It is initialised once at startup via NewValidator; per-message checks are cheap.
type TopicValidator struct {
	ExpectedID      uint32
	ExpectedSubject string
}

// NewValidator fetches the latest schema ID for subject from registryURL using the
// Confluent-compat v7 API (GET /subjects/{subject}/versions/latest).
//
// Fails fast if Apicurio is unreachable — treat the returned error as fatal at startup.
// The schema ID is pinned for the lifetime of the process; restart to pick up a new schema.
func NewValidator(registryURL, subject string) (*TopicValidator, error) {
	id, err := fetchLatestSchemaID(registryURL, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to sync schema from Apicurio (subject=%s): %w", subject, err)
	}
	return &TopicValidator{ExpectedID: uint32(id), ExpectedSubject: subject}, nil
}

// ValidateAndUnwrap checks the Confluent Wire Format prefix and returns the raw
// Protobuf bytes (without the 5-byte header and message index).
//
// Confluent Protobuf Wire Format appends message index bytes after the 5-byte
// header: uvarint(count) followed by count zigzag varints. count=0 (0x00) means
// "first message in the schema" — the common case written by RisingWave.
//
// On mismatch the caller should route the record to the DLQ rather than returning
// an error that would crash the consumer loop.
func (v *TopicValidator) ValidateAndUnwrap(data []byte) ([]byte, error) {
	if len(data) < confluentHeaderSize || data[0] != confluentMagicByte {
		return nil, errors.New("invalid confluent wire format: missing magic byte or too short")
	}
	msgID := binary.BigEndian.Uint32(data[1:confluentHeaderSize])
	if msgID != v.ExpectedID {
		return nil, fmt.Errorf("schema mismatch: expected ID %d (%s), got %d",
			v.ExpectedID, v.ExpectedSubject, msgID)
	}
	rest := data[confluentHeaderSize:]
	n, err := skipMessageIndexes(rest)
	if err != nil {
		return nil, err
	}
	return rest[n:], nil
}

// skipMessageIndexes advances past the Confluent Protobuf message index bytes
// that follow the 5-byte Confluent Wire Format header.
//
// Format: uvarint(count), then count zigzag-encoded varints.
// count=0 is a shorthand for "first/only message" and occupies exactly one byte (0x00).
// Returns the number of bytes consumed.
func skipMessageIndexes(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, errors.New("confluent protobuf: missing message index")
	}
	count, n := binary.Uvarint(data)
	if n <= 0 {
		return 0, errors.New("confluent protobuf: invalid message index varint")
	}
	offset := n
	for i := uint64(0); i < count; i++ {
		_, n = binary.Uvarint(data[offset:])
		if n <= 0 {
			return 0, errors.New("confluent protobuf: truncated message index")
		}
		offset += n
	}
	return offset, nil
}

// Unwrap validates the Confluent Wire Format magic byte and returns the schema ID
// and raw Protobuf bytes without enforcing a specific expected schema ID.
// Use this when the topic carries multiple schema versions (e.g. v1 and v2).
func Unwrap(data []byte) (schemaID uint32, protoBytes []byte, err error) {
	if len(data) < confluentHeaderSize || data[0] != confluentMagicByte {
		return 0, nil, errors.New("invalid confluent wire format: missing magic byte or too short")
	}
	id := binary.BigEndian.Uint32(data[1:confluentHeaderSize])
	rest := data[confluentHeaderSize:]
	n, err := skipMessageIndexes(rest)
	if err != nil {
		return 0, nil, err
	}
	return id, rest[n:], nil
}

// fetchLatestSchemaID queries the Confluent-compat v7 API for the latest schema ID.
func fetchLatestSchemaID(registryURL, subject string) (int, error) {
	url := strings.TrimSuffix(registryURL, "/") + "/subjects/" + subject + "/versions/latest"
	client := &http.Client{Timeout: 10 * time.Second}

	once := func() (int, error) {
		resp, err := client.Get(url) //nolint:gosec // URL is from internal config, not user input
		if err != nil {
			return 0, fmt.Errorf("http get %s: %w", url, err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("registry returned HTTP %d for subject %q", resp.StatusCode, subject)
		}
		var body struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return 0, fmt.Errorf("decode registry response: %w", err)
		}
		if body.ID == 0 {
			return 0, fmt.Errorf("registry returned schema id 0 for subject %q", subject)
		}
		return body.ID, nil
	}

	var lastErr error
	for i := range schemaFetchMaxAttempts {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}
		if id, err := once(); err == nil {
			return id, nil
		} else {
			lastErr = err
		}
	}
	return 0, lastErr
}
