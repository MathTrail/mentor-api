package onboarding

import (
	"context"
	"fmt"
	"time"

	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	"github.com/google/uuid"
	studentsv1 "github.com/mathtrail/contracts/gen/go/students/v1"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const topic = "students.onboarding.ready"

var tracer = otel.Tracer("mentor-api/onboarding")

// Consumer reads from students.onboarding.ready and persists recommendations.
type Consumer struct {
	client    *kgo.Client
	repo      *Repository
	validator *kafkainfra.TopicValidator
	logger    *zap.Logger
}

func NewConsumer(client *kgo.Client, repo *Repository, validator *kafkainfra.TopicValidator, logger *zap.Logger) *Consumer {
	return &Consumer{
		client:    client,
		repo:      repo,
		validator: validator,
		logger:    logger,
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

	rawProto, err := c.validator.ValidateAndUnwrap(record.Value)
	if err != nil {
		// Poison pill — route to DLQ, do not crash the consumer loop.
		c.produceDLQ(record)
		c.logger.Warn("schema validation failed, routed to DLQ",
			zap.Error(err),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}

	var msg studentsv1.StudentOnboardingReady
	if err := proto.Unmarshal(rawProto, &msg); err != nil {
		c.produceDLQ(record)
		c.logger.Warn("proto unmarshal failed, routed to DLQ",
			zap.Error(err),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}

	ce := kafkainfra.ExtractCloudEventHeaders(record.Headers)

	studentID, err := uuid.Parse(msg.UserId)
	if err != nil {
		c.produceDLQ(record)
		c.logger.Warn("invalid user_id, routed to DLQ",
			zap.Error(err),
			zap.String("user_id", msg.UserId),
		)
		return nil
	}

	occurredAt, err := parseOccurredAt(msg.OccurredAt)
	if err != nil {
		c.produceDLQ(record)
		c.logger.Warn("invalid occurred_at, routed to DLQ",
			zap.Error(err),
			zap.String("occurred_at", msg.OccurredAt),
		)
		return nil
	}

	span.SetAttributes(
		attribute.String("event.id", msg.EventId),
		attribute.String("student.id", msg.UserId),
		attribute.String("event.occurred_at", msg.OccurredAt),
		attribute.String("ce.id", ce.ID),
		attribute.String("ce.type", ce.Type),
	)

	if err := c.repo.Upsert(ctx, studentID, msg.EventId, occurredAt); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("upsert: %w", err)
	}

	c.logger.Info("recommendation upserted",
		zap.String("student_id", msg.UserId),
		zap.String("event_id", msg.EventId),
		zap.String("ce_id", ce.ID),
	)
	return nil
}

// produceDLQ forwards the original record (value + headers) to the DLQ topic.
// CloudEvents headers are preserved for end-to-end tracing.
// A 5-second timeout prevents the produce attempt from blocking indefinitely
// if the broker is unavailable; the callback logs any delivery failure.
func (c *Consumer) produceDLQ(record *kgo.Record) {
	dlqRecord := &kgo.Record{
		Topic:   record.Topic + ".dlq",
		Value:   record.Value,
		Headers: record.Headers,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	c.client.Produce(ctx, dlqRecord, func(_ *kgo.Record, err error) {
		cancel()
		if err != nil {
			c.logger.Error("failed to produce to DLQ",
				zap.Error(err),
				zap.String("dlq_topic", dlqRecord.Topic),
			)
		}
	})
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
