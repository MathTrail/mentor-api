package kafka

import "github.com/twmb/franz-go/pkg/kgo"

// RecordCarrier adapts a slice of kgo.RecordHeader to the OTel TextMapCarrier
// interface, enabling W3C TraceContext propagation across Kafka records.
type RecordCarrier struct {
	headers []kgo.RecordHeader
}

// NewRecordCarrier wraps the record's headers for use with otel.GetTextMapPropagator().
func NewRecordCarrier(headers []kgo.RecordHeader) *RecordCarrier {
	return &RecordCarrier{headers: headers}
}

func (c *RecordCarrier) Get(key string) string {
	for _, h := range c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *RecordCarrier) Set(key string, value string) {
	c.headers = append(c.headers, kgo.RecordHeader{Key: key, Value: []byte(value)})
}

func (c *RecordCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, h := range c.headers {
		keys[i] = h.Key
	}
	return keys
}
