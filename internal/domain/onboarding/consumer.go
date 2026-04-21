package onboarding

import (
	"context"
	"fmt"
	"time"

	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	"github.com/google/uuid"
	studentsv1 "github.com/mathtrail/contracts/gen/go/students/v1"
	studentsv2 "github.com/mathtrail/contracts/gen/go/students/v2"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const topic = "students.onboarding.ready"

var tracer = otel.Tracer("mentor-api/onboarding")

// DLQPublisher routes unprocessable records to a dead-letter queue.
type DLQPublisher interface {
	PublishDLQ(record *kgo.Record)
}

// Consumer reads from students.onboarding.ready and persists recommendations.
// Handles both v1 (students.v1.StudentOnboardingReady) and
// v2 (students.v2.StudentOnboardingReady) schemas on the same topic,
// distinguished by the Confluent Wire Format schema ID prefix.
type Consumer struct {
	client     *kgo.Client
	dlq        DLQPublisher
	repo       *Repository
	v1SchemaID uint32
	v2SchemaID uint32
	logger     *zap.Logger
}

func (c *Consumer) expectedSchemaIDs() []uint32 {
	return []uint32{c.v1SchemaID, c.v2SchemaID}
}

func NewConsumer(
	client *kgo.Client,
	repo *Repository,
	v1Validator *kafkainfra.TopicValidator,
	v2Validator *kafkainfra.TopicValidator,
	logger *zap.Logger,
) *Consumer {
	return &Consumer{
		client:     client,
		dlq:        &kafkaDLQPublisher{client: client, logger: logger},
		repo:       repo,
		v1SchemaID: v1Validator.ExpectedID,
		v2SchemaID: v2Validator.ExpectedID,
		logger:     logger,
	}
}

// Start runs the consume loop until ctx is cancelled.
// Blocks the caller — intended to be run via errgroup or a managed goroutine.
// Returns nil on clean shutdown (ctx cancelled or client closed).
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("starting onboarding consumer", zap.String("topic", topic))
	c.client.AddConsumeTopics(topic)

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			break
		}

		fetches.EachError(func(t string, p int32, err error) {
			c.logger.Error("kafka fetch error",
				zap.String("topic", t),
				zap.Int32("partition", p),
				zap.Error(err),
			)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			if err := c.handle(ctx, record); err != nil {
				c.logger.Error("failed to handle onboarding event",
					zap.Error(err),
					zap.String("topic", record.Topic),
					zap.Int32("partition", record.Partition),
					zap.Int64("offset", record.Offset),
				)
			}
		})
	}

	// Graceful shutdown: leave the consumer group so Kafka can reassign partitions
	// immediately instead of waiting for the session timeout.
	// Close() is intentionally omitted — the kgo.Client is owned by the container
	// and will be closed after all workers have stopped.
	c.logger.Info("onboarding consumer shutting down, leaving group")
	c.client.LeaveGroup()
	c.logger.Info("onboarding consumer stopped")
	return nil
}

func (c *Consumer) handle(ctx context.Context, record *kgo.Record) error {
	// Propagate W3C traceparent from Kafka headers to link producer and consumer traces.
	ctx = otel.GetTextMapPropagator().Extract(ctx, kafkainfra.NewRecordCarrier(record.Headers))
	ctx, span := tracer.Start(ctx, "onboarding.recommendation.upsert")
	defer span.End()

	schemaID, rawProto, err := kafkainfra.Unwrap(record.Value)
	if err != nil {
		c.dlq.PublishDLQ(record)
		c.logger.Warn("wire format invalid, routed to DLQ",
			zap.Error(err),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}

	switch schemaID {
	case c.v1SchemaID:
		return c.handleV1(ctx, span, record, rawProto)
	case c.v2SchemaID:
		return c.handleV2(ctx, span, record, rawProto)
	default:
		return c.handleUnknownSchema(ctx, span, record, schemaID, rawProto)
	}
}

// handleUnknownSchema provides forward-compatibility when a new registry ID is
// introduced for a payload that is still compatible with v1/v2 contracts.
// If payload decoding fails for both contracts, the record is routed to DLQ.
func (c *Consumer) handleUnknownSchema(
	ctx context.Context,
	span trace.Span,
	record *kgo.Record,
	schemaID uint32,
	rawProto []byte,
) error {
	var msgV2 studentsv2.StudentOnboardingReady
	if err := proto.Unmarshal(rawProto, &msgV2); err == nil && msgV2.EventId != "" && msgV2.UserId != "" && msgV2.OccurredAt != "" {
		c.logger.Info("unknown schema ID accepted as v2-compatible payload",
			zap.Uint32("schema_id", schemaID),
			zap.Uint32s("expected_schema_ids", c.expectedSchemaIDs()),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return c.process(ctx, span, record, msgV2.EventId, msgV2.UserId, msgV2.OccurredAt)
	}

	var msgV1 studentsv1.StudentOnboardingReady
	if err := proto.Unmarshal(rawProto, &msgV1); err == nil && msgV1.EventId != "" && msgV1.UserId != "" && msgV1.OccurredAt != "" {
		c.logger.Info("unknown schema ID accepted as v1-compatible payload",
			zap.Uint32("schema_id", schemaID),
			zap.Uint32s("expected_schema_ids", c.expectedSchemaIDs()),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return c.process(ctx, span, record, msgV1.EventId, msgV1.UserId, msgV1.OccurredAt)
	}

	c.dlq.PublishDLQ(record)
	c.logger.Warn("unknown schema ID, routed to DLQ",
		zap.Uint32("schema_id", schemaID),
		zap.Uint32s("expected_schema_ids", c.expectedSchemaIDs()),
		zap.Int32("partition", record.Partition),
		zap.Int64("offset", record.Offset),
	)
	return nil
}

func (c *Consumer) handleV1(ctx context.Context, span trace.Span, record *kgo.Record, rawProto []byte) error {
	var msg studentsv1.StudentOnboardingReady
	if err := proto.Unmarshal(rawProto, &msg); err != nil {
		c.dlq.PublishDLQ(record)
		c.logger.Warn("proto v1 unmarshal failed, routed to DLQ",
			zap.Error(err),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}
	return c.process(ctx, span, record, msg.EventId, msg.UserId, msg.OccurredAt)
}

func (c *Consumer) handleV2(ctx context.Context, span trace.Span, record *kgo.Record, rawProto []byte) error {
	var msg studentsv2.StudentOnboardingReady
	if err := proto.Unmarshal(rawProto, &msg); err != nil {
		c.dlq.PublishDLQ(record)
		c.logger.Warn("proto v2 unmarshal failed, routed to DLQ",
			zap.Error(err),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}
	return c.process(ctx, span, record, msg.EventId, msg.UserId, msg.OccurredAt)
}

func (c *Consumer) process(ctx context.Context, span trace.Span, record *kgo.Record, eventID, userID, occurredAtRaw string) error {
	ce := kafkainfra.ExtractCloudEventHeaders(record.Headers)

	studentID, err := uuid.Parse(userID)
	if err != nil {
		c.dlq.PublishDLQ(record)
		c.logger.Warn("invalid user_id, routed to DLQ",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return nil
	}

	occurredAt, err := parseOccurredAt(occurredAtRaw)
	if err != nil {
		c.dlq.PublishDLQ(record)
		c.logger.Warn("invalid occurred_at, routed to DLQ",
			zap.Error(err),
			zap.String("occurred_at", occurredAtRaw),
		)
		return nil
	}

	span.SetAttributes(
		attribute.String("event.id", eventID),
		attribute.String("student.id", userID),
		attribute.String("event.occurred_at", occurredAtRaw),
		attribute.String("ce.id", ce.ID),
		attribute.String("ce.type", ce.Type),
	)

	if err := c.repo.Upsert(ctx, studentID, eventID, occurredAt); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("upsert: %w", err)
	}

	c.logger.Info("recommendation upserted",
		zap.String("student_id", userID),
		zap.String("event_id", eventID),
		zap.String("ce_id", ce.ID),
	)
	return nil
}

// kafkaDLQPublisher forwards unprocessable records to the DLQ topic via Kafka.
// CloudEvents headers are preserved for end-to-end tracing.
// A 5-second timeout prevents the produce attempt from blocking indefinitely
// if the broker is unavailable; delivery failures are logged.
type kafkaDLQPublisher struct {
	client *kgo.Client
	logger *zap.Logger
}

func (p *kafkaDLQPublisher) PublishDLQ(record *kgo.Record) {
	dlqRecord := &kgo.Record{
		Topic:   record.Topic + ".dlq",
		Value:   record.Value,
		Headers: record.Headers,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.client.ProduceSync(ctx, dlqRecord).FirstErr(); err != nil {
		p.logger.Error("failed to produce to DLQ",
			zap.Error(err),
			zap.String("dlq_topic", dlqRecord.Topic),
		)
	}
}

// parseOccurredAt tries RFC3339 first, then RisingWave's NOW()::VARCHAR format.
func parseOccurredAt(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp format: %q", s)
}
