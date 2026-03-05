package kafka

// Implement Kafka client wrapper (sarama or confluent-kafka-go).
// Responsibilities:
//   - Client initialisation with TLS/SASL from config
//   - Producer and consumer group construction
//   - Retry and backoff configuration
