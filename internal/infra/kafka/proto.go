package kafka

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// confluentMagicByte is the first byte of every Confluent Wire Format message (always 0x00).
// confluentHeaderSize is the size of the Confluent Wire Format header: 1 magic byte + 4 schema ID bytes.
const (
	confluentMagicByte  = 0x00
	confluentHeaderSize = 5
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
// Protobuf bytes (without the 5-byte header).
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
	return data[confluentHeaderSize:], nil
}

// fetchLatestSchemaID queries the Confluent-compat v7 API for the latest schema ID.
func fetchLatestSchemaID(registryURL, subject string) (int, error) {
	url := registryURL + "/subjects/" + subject + "/versions/latest"
	resp, err := http.Get(url) //nolint:gosec // URL is from internal config, not user input
	if err != nil {
		return 0, fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()
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
