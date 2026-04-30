package onboarding

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	onboardingmocks "github.com/MathTrail/mentor-api/internal/domain/onboarding/mocks"
	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	pgmocks "github.com/MathTrail/mentor-api/internal/infra/postgres/mocks"
	"github.com/google/uuid"
	studentsv1 "github.com/mathtrail/contracts/gen/go/students/v1"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// --- parseOccurredAt ---

func TestParseOccurredAt_RFC3339Nano(t *testing.T) {
	ts, err := parseOccurredAt("2024-01-15T10:30:00.123456789Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Nanosecond() != 123456789 {
		t.Errorf("nanoseconds: got %d, want 123456789", ts.Nanosecond())
	}
}

func TestParseOccurredAt_RFC3339(t *testing.T) {
	if _, err := parseOccurredAt("2024-01-15T10:30:00Z"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOccurredAt_RisingWaveNano(t *testing.T) {
	if _, err := parseOccurredAt("2024-01-15T10:30:00.123456789"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOccurredAt_RisingWaveSpace(t *testing.T) {
	if _, err := parseOccurredAt("2024-01-15 10:30:00.123456789"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOccurredAt_RisingWavePlain(t *testing.T) {
	if _, err := parseOccurredAt("2024-01-15 10:30:00"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOccurredAt_Invalid(t *testing.T) {
	ts, err := parseOccurredAt("not-a-date")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time, got %v", ts)
	}
}

// --- handle() helpers ---

// wireRecord builds a kgo.Record whose Value is a Confluent Protobuf Wire Format message:
// 5-byte header (magic + schema ID) + 1-byte message index (0x00) + protobuf payload.
func wireRecord(schemaID uint32, protoBytes []byte) *kgo.Record {
	buf := make([]byte, 6+len(protoBytes))
	buf[0] = 0x00
	binary.BigEndian.PutUint32(buf[1:5], schemaID)
	buf[5] = 0x00 // message index: count=0 (first/only message)
	copy(buf[6:], protoBytes)
	return &kgo.Record{Topic: "students.onboarding.ready", Value: buf}
}

// marshalProto marshals a StudentOnboardingReady and panics if it fails (test helper).
func marshalProto(t *testing.T, msg *studentsv1.StudentOnboardingReady) []byte {
	t.Helper()
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return b
}

// newTestConsumer builds a Consumer with mock DLQ + real Repository backed by mock DB.
// The TopicValidator is pre-set to ExpectedID=42 (no network call needed).
func newTestConsumer(t *testing.T, dlq *onboardingmocks.MockDLQPublisher, db *pgmocks.MockDB) *Consumer {
	t.Helper()
	return &Consumer{
		client:    nil, // not called in handle()
		dlq:       dlq,
		repo:      NewRepository(db),
		validator: &kafkainfra.TopicValidator{ExpectedID: 42, ExpectedSubject: "students.onboarding.ready"},
		logger:    zap.NewNop(),
	}
}

// --- handle() tests ---

func TestHandle_InvalidWireFormat(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	dlq.EXPECT().PublishDLQ(mock.Anything).Return()
	db := pgmocks.NewMockDB(t)

	c := newTestConsumer(t, dlq, db)
	record := &kgo.Record{Value: []byte("garbage")}
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("expected nil (DLQ route), got: %v", err)
	}
}

func TestHandle_SchemaMismatch(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	dlq.EXPECT().PublishDLQ(mock.Anything).Return()
	db := pgmocks.NewMockDB(t)

	c := newTestConsumer(t, dlq, db)
	// Valid wire format but schema ID 99 ≠ expected 42.
	record := wireRecord(99, []byte("payload"))
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("expected nil (DLQ route), got: %v", err)
	}
}

func TestHandle_InvalidProto(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	dlq.EXPECT().PublishDLQ(mock.Anything).Return()
	db := pgmocks.NewMockDB(t)

	c := newTestConsumer(t, dlq, db)
	// Correct 5-byte wire header, but proto body is garbage.
	record := wireRecord(42, []byte{0xFF, 0xFE, 0xFD})
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("expected nil (DLQ route), got: %v", err)
	}
}

func TestHandle_InvalidUserID(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	dlq.EXPECT().PublishDLQ(mock.Anything).Return()
	db := pgmocks.NewMockDB(t)

	c := newTestConsumer(t, dlq, db)
	msg := &studentsv1.StudentOnboardingReady{
		EventId:    "evt-1",
		UserId:     "not-a-uuid",
		OccurredAt: "2024-01-15T10:30:00Z",
	}
	record := wireRecord(42, marshalProto(t, msg))
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("expected nil (DLQ route), got: %v", err)
	}
}

func TestHandle_InvalidOccurredAt(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	dlq.EXPECT().PublishDLQ(mock.Anything).Return()
	db := pgmocks.NewMockDB(t)

	c := newTestConsumer(t, dlq, db)
	msg := &studentsv1.StudentOnboardingReady{
		EventId:    "evt-2",
		UserId:     uuid.New().String(),
		OccurredAt: "bad-timestamp",
	}
	record := wireRecord(42, marshalProto(t, msg))
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("expected nil (DLQ route), got: %v", err)
	}
}

func TestHandle_UpsertError(t *testing.T) {
	upsertErr := errors.New("db down")
	dlq := onboardingmocks.NewMockDLQPublisher(t) // no DLQ call expected
	db := pgmocks.NewMockDB(t)
	db.EXPECT().Exec(
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything,
	).Return(upsertErr)

	c := newTestConsumer(t, dlq, db)
	msg := &studentsv1.StudentOnboardingReady{
		EventId:    "evt-3",
		UserId:     uuid.New().String(),
		OccurredAt: "2024-01-15T10:30:00Z",
	}
	record := wireRecord(42, marshalProto(t, msg))
	err := c.handle(context.Background(), record)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, upsertErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, upsertErr)
	}
}

func TestHandle_Success(t *testing.T) {
	dlq := onboardingmocks.NewMockDLQPublisher(t)
	db := pgmocks.NewMockDB(t)
	db.EXPECT().Exec(
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	c := newTestConsumer(t, dlq, db)
	msg := &studentsv1.StudentOnboardingReady{
		EventId:    "evt-4",
		UserId:     uuid.New().String(),
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}
	record := wireRecord(42, marshalProto(t, msg))
	if err := c.handle(context.Background(), record); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
func TestNewConsumer(t *testing.T) {
	validator := &kafkainfra.TopicValidator{ExpectedID: 99, ExpectedSubject: "test-subject"}
	c := NewConsumer(nil, nil, validator, zap.NewNop())
	if c == nil {
		t.Fatal("expected non-nil consumer")
	}
	if c.validator != validator {
		t.Errorf("validator not set: got %v, want %v", c.validator, validator)
	}
}
