package kafka

import (
	"fmt"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"go.uber.org/zap"

	"github.com/MathTrail/mentor-api/internal/config"
)

// ClientConfig holds all parameters needed to create a franz-go Kafka client.
type ClientConfig struct {
	BootstrapServers []string
	ConsumerGroup    string
	// InstanceID is used for static group membership (set to POD_NAME).
	// Static membership prevents rebalance storms during rolling deploys.
	InstanceID   string
	SASLUsername string
	SASLPassword string
}

// NewClientFromConfig creates a Kafka client from application config.
// It validates that at least one broker address is present and wires the
// provided logger into the client so kafka-level events are visible in app logs.
func NewClientFromConfig(cfg *config.Config, logger *zap.Logger) (*kgo.Client, error) {
	var brokers []string
	for _, b := range strings.Split(cfg.KafkaBootstrapServers, ",") {
		if addr := strings.TrimSpace(b); addr != "" {
			brokers = append(brokers, addr)
		}
	}
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka: KAFKA_BOOTSTRAP_SERVERS is empty")
	}

	return NewClient(
		ClientConfig{
			BootstrapServers: brokers,
			ConsumerGroup:    cfg.KafkaConsumerGroup,
			InstanceID:       cfg.InstanceID,
			SASLUsername:     cfg.KafkaSASLUsername,
			SASLPassword:     cfg.KafkaSASLPassword,
		},
		kgo.WithLogger(newZapLogger(logger)),
	)
}

// NewClient creates a franz-go Kafka client with SASL/SCRAM-SHA-512 and static group membership.
func NewClient(cfg ClientConfig, extraOpts ...kgo.Opt) (*kgo.Client, error) {
	auth := scram.Auth{
		User: cfg.SASLUsername,
		Pass: cfg.SASLPassword,
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.BootstrapServers...),
		kgo.SASL(auth.AsSha512Mechanism()),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.InstanceID(cfg.InstanceID),
	}
	opts = append(opts, extraOpts...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka new client: %w", err)
	}
	return client, nil
}
