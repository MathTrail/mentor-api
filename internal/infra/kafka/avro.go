package kafka

import (
	"encoding/binary"
	"fmt"
)

// Confluent wire format: [0x00 magic][schema_id: 4 bytes big-endian][avro binary payload]
// Avro binary string encoding: zigzag-varint length followed by UTF-8 bytes.
//
// StudentOnboardingReadyPayload matches the students.onboarding.ready-value schema:
//   {event_id, user_id, email, first_name, last_name, role, occurred_at} — all strings
// Field order is fixed by the schema (Avro records are positional in binary format).

// StudentOnboardingReadyPayload is the decoded domain event from students.onboarding.ready.
type StudentOnboardingReadyPayload struct {
	EventID    string
	UserID     string
	Email      string
	FirstName  string
	LastName   string
	Role       string
	OccurredAt string
}

// DecodeStudentOnboardingReady strips the Confluent 5-byte header and decodes the
// Avro binary payload into StudentOnboardingReadyPayload.
// All fields in the schema are strings (no unions, no nested types).
func DecodeStudentOnboardingReady(data []byte) (*StudentOnboardingReadyPayload, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}
	if data[0] != 0x00 {
		return nil, fmt.Errorf("invalid magic byte: 0x%02x (expected 0x00)", data[0])
	}
	// Skip magic byte + 4-byte schema ID
	_ = int32(binary.BigEndian.Uint32(data[1:5]))
	buf := data[5:]

	fields := make([]string, 7)
	var err error
	for i := range fields {
		fields[i], buf, err = readAvroString(buf)
		if err != nil {
			return nil, fmt.Errorf("field %d: %w", i, err)
		}
	}

	return &StudentOnboardingReadyPayload{
		EventID:    fields[0],
		UserID:     fields[1],
		Email:      fields[2],
		FirstName:  fields[3],
		LastName:   fields[4],
		Role:       fields[5],
		OccurredAt: fields[6],
	}, nil
}

// readAvroString reads a single Avro-encoded string from buf.
// Returns the string, the remaining bytes, and any error.
func readAvroString(buf []byte) (string, []byte, error) {
	n, bytesRead, err := readZigzagLong(buf)
	if err != nil {
		return "", buf, fmt.Errorf("read string length: %w", err)
	}
	buf = buf[bytesRead:]
	if n < 0 {
		return "", buf, fmt.Errorf("negative string length: %d", n)
	}
	if int(n) > len(buf) {
		return "", buf, fmt.Errorf("string length %d exceeds buffer %d", n, len(buf))
	}
	return string(buf[:n]), buf[n:], nil
}

// readZigzagLong decodes a zigzag-encoded varint from buf.
// Returns the decoded int64, number of bytes consumed, and any error.
func readZigzagLong(buf []byte) (int64, int, error) {
	var n uint64
	var shift uint
	for i, b := range buf {
		n |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			// Zigzag decode: (n >>> 1) ^ -(n & 1)
			decoded := int64((n >> 1) ^ -(n & 1))
			return decoded, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, fmt.Errorf("varint overflow")
		}
	}
	return 0, 0, fmt.Errorf("unexpected end of buffer reading varint")
}
