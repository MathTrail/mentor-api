package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"go.uber.org/zap"

	"github.com/MathTrail/mentor-api/internal/logger"
)

// topicSpec holds the desired configuration for a single Kafka topic.
type topicSpec struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
}

// config holds all startup configuration resolved from environment variables.
type config struct {
	BootstrapServers string
	Topics           []topicSpec
	SASLUser         string
	SASLPass         string
}

// loadConfig reads and validates all required environment variables.
// Calls log.Fatal on the first missing or malformed value.
func loadConfig(log *zap.Logger) config {
	bootstrapServers, err := requiredEnv("KAFKA_BOOTSTRAP_SERVERS")
	if err != nil {
		log.Fatal("missing required config", zap.Error(err))
	}
	topicsRaw, err := requiredEnv("KAFKA_TOPICS")
	if err != nil {
		log.Fatal("missing required config", zap.Error(err))
	}
	topics, err := parseTopics(topicsRaw)
	if err != nil {
		log.Fatal("invalid KAFKA_TOPICS", zap.Error(err))
	}
	return config{
		BootstrapServers: bootstrapServers,
		Topics:           topics,
		SASLUser:         envOrDefault("KAFKA_SASL_USERNAME", ""),
		SASLPass:         envOrDefault("KAFKA_SASL_PASSWORD", ""),
	}
}

func main() {
	log := logger.NewLogger("kafka-setup", "info", "json")

	cfg := loadConfig(log)

	brokers := splitBrokers(cfg.BootstrapServers)

	opts := []kgo.Opt{kgo.SeedBrokers(brokers...)}

	// SASL is opt-in: only enabled when both credentials are provided.
	// Brokers configured with PLAINTEXT listeners (e.g. local dev AutoMQ on port 9092)
	// reject any SASL handshake with ILLEGAL_SASL_STATE. Keeping SASL unconditional
	// would break plaintext-only environments even when credentials are empty strings.
	if cfg.SASLUser != "" && cfg.SASLPass != "" {
		auth := scram.Auth{User: cfg.SASLUser, Pass: cfg.SASLPass}
		opts = append(opts, kgo.SASL(auth.AsSha512Mechanism()))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatal("failed to create kafka client", zap.Error(err))
	}
	defer client.Close()

	adm := kadm.NewClient(client)

	// Retry loop: AutoMQ controller may not be fully initialised at the time
	// this Job starts (especially during first cluster boot or CI). Transient
	// network errors are retried up to 3 times with a 2-second pause.
	const (
		maxAttempts = 3
		retryPause  = 2 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = createTopics(ctx, adm, cfg.Topics, log)
		if lastErr == nil {
			log.Info("kafka topics ready")
			return
		}

		// Do not retry on configuration or auth errors — only transient ones.
		if !isTransient(lastErr) {
			log.Fatal("permanent error creating topics", zap.Error(lastErr))
		}

		if attempt < maxAttempts {
			log.Warn("transient error creating topics, retrying",
				zap.Error(lastErr),
				zap.Int("attempt", attempt),
				zap.Duration("retry_in", retryPause),
			)
			select {
			case <-time.After(retryPause):
				// continue retry loop
			case <-ctx.Done():
				log.Fatal("context expired while waiting to retry", zap.Error(ctx.Err()))
			}
		}
	}

	log.Fatal("failed to create topics after retries", zap.Error(lastErr))
}

// createTopics creates all specified topics. TOPIC_ALREADY_EXISTS is treated as
// success — this makes the Job safe to run on every deploy (idempotent).
func createTopics(ctx context.Context, adm *kadm.Client, topics []topicSpec, log *zap.Logger) error {
	for _, t := range topics {
		resp, err := adm.CreateTopics(ctx, t.Partitions, t.ReplicationFactor, nil, t.Name)
		if err != nil {
			return fmt.Errorf("create topic %q: %w", t.Name, err)
		}
		for _, r := range resp {
			if r.Err != nil && !errors.Is(r.Err, kerr.TopicAlreadyExists) {
				return fmt.Errorf("create topic %q: %w", r.Topic, r.Err)
			}
			if errors.Is(r.Err, kerr.TopicAlreadyExists) {
				log.Debug("topic already exists, skipping", zap.String("topic", r.Topic))
			} else {
				log.Info("topic created", zap.String("topic", r.Topic),
					zap.Int32("partitions", t.Partitions),
					zap.Int16("replication_factor", t.ReplicationFactor),
				)
			}
		}
	}
	return nil
}

// parseTopics parses the KAFKA_TOPICS env var value.
// Format: "name:partitions:replication[,name:partitions:replication,...]"
// Example: "students.onboarding.ready.dlq:1:1"
func parseTopics(raw string) ([]topicSpec, error) {
	var topics []topicSpec
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("entry %q must have format name:partitions:replication", entry)
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			return nil, fmt.Errorf("entry %q has empty topic name", entry)
		}
		partitions, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil || partitions <= 0 {
			return nil, fmt.Errorf("entry %q has invalid partitions %q: must be a positive integer", entry, parts[1])
		}
		replication, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 16)
		if err != nil || replication <= 0 {
			return nil, fmt.Errorf("entry %q has invalid replication factor %q: must be a positive integer", entry, parts[2])
		}
		topics = append(topics, topicSpec{
			Name:              name,
			Partitions:        int32(partitions),
			ReplicationFactor: int16(replication),
		})
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("KAFKA_TOPICS is empty or contains no valid entries")
	}
	return topics, nil
}

// isTransient returns true for errors that are worth retrying (network-level
// failures). Auth errors, invalid configs, and Kafka protocol errors that
// indicate a permanent condition are not retried.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	// Prefer typed checks over string matching: net.Error covers dial failures,
	// connection refused, i/o timeouts, and DNS errors.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// io.EOF can occur when the broker closes the connection before responding.
	return errors.Is(err, io.EOF)
}

func splitBrokers(s string) []string {
	var brokers []string
	for _, b := range strings.Split(s, ",") {
		if addr := strings.TrimSpace(b); addr != "" {
			brokers = append(brokers, addr)
		}
	}
	return brokers
}

// requiredEnv reads an environment variable or returns an error if it is empty.
// A missing variable means the K8s Job is misconfigured — fail fast.
func requiredEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
