package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafkainfra "github.com/MathTrail/mentor-api/internal/infra/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

const topic = "students.onboarding.ready"

var tracer = otel.Tracer("mentor-api/onboarding")

// Consumer reads from students.onboarding.ready and persists recommendations.
type Consumer struct {
	client *kgo.Client
	repo   *Repository
	logger *zap.Logger
}

func NewConsumer(client *kgo.Client, repo *Repository, logger *zap.Logger) *Consumer {
	return &Consumer{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// Start runs the consume loop until ctx is cancelled.
// Blocks the caller — run in a goroutine.
func (c *Consumer) Start(ctx context.Context) {
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
	c.logger.Info("onboarding consumer shutting down, leaving group")
	c.client.LeaveGroup()
	c.client.Close()
	c.logger.Info("onboarding consumer stopped")
}

func (c *Consumer) handle(ctx context.Context, record *kgo.Record) error {
	ctx, span := tracer.Start(ctx, "onboarding.recommendation.upsert")
	defer span.End()

	event, err := kafkainfra.DecodeStudentOnboardingReady(record.Value)
	if err != nil {
		return fmt.Errorf("decode avro: %w", err)
	}

	studentID, err := uuid.Parse(event.UserID)
	if err != nil {
		return fmt.Errorf("parse user_id %q: %w", event.UserID, err)
	}

	occurredAt, err := parseOccurredAt(event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurred_at %q: %w", event.OccurredAt, err)
	}

	span.SetAttributes(
		attribute.String("event.id", event.EventID),
		attribute.String("student.id", event.UserID),
		attribute.String("event.occurred_at", event.OccurredAt),
	)

	if err := c.repo.Upsert(ctx, studentID, event.EventID, occurredAt); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("upsert: %w", err)
	}

	c.logger.Info("recommendation upserted",
		zap.String("student_id", event.UserID),
		zap.String("event_id", event.EventID),
	)
	return nil
}

// parseOccurredAt tries RFC3339 first, then Flink's CAST(event_time AS STRING) format.
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
