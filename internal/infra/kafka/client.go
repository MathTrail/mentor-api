package kafka

import (
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
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
		// Never auto-create topics — fail fast on misconfiguration
		kgo.AllowAutoTopicCreation(false),
	}
	opts = append(opts, extraOpts...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka new client: %w", err)
	}
	return client, nil
}
