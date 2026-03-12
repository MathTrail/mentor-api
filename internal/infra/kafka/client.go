package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
)

// ClientConfig holds all parameters needed to create a franz-go Kafka client.
type ClientConfig struct {
	BootstrapServers []string
	ConsumerGroup    string
	// InstanceID is used for static group membership (set to POD_NAME).
	// Static membership prevents rebalance storms during rolling deploys.
	InstanceID string
	// TLSCertDir is the directory containing user.crt, user.key, ca.crt
	// (mounted from the Strimzi KafkaUser secret via VSO or direct mount).
	TLSCertDir string
}

// NewClient creates a franz-go Kafka client with mTLS and static group membership.
func NewClient(cfg ClientConfig, extraOpts ...kgo.Opt) (*kgo.Client, error) {
	tlsCfg, err := buildTLSConfig(cfg.TLSCertDir)
	if err != nil {
		return nil, fmt.Errorf("kafka tls config: %w", err)
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.BootstrapServers...),
		kgo.DialTLSConfig(tlsCfg),
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

func buildTLSConfig(certDir string) (*tls.Config, error) {
	certFile := certDir + "/user.crt"
	keyFile := certDir + "/user.key"
	caFile := certDir + "/ca.crt"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
