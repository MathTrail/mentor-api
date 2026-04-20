package kafka

import "github.com/twmb/franz-go/pkg/kgo"

// CloudEventHeaders holds the CloudEvents Binary Mode attributes extracted from
// Kafka record headers. Attributes are written by RisingWave sink via the
// _headers MAP column.
//
// Standard CloudEvents Kafka Binary Content Mode header names use dashes (ce-type, ce-id, etc.)
// per the CloudEvents Kafka Protocol Binding specification.
//
// Missing headers produce empty strings — records without CE attributes are
// still processed; downstream callers should treat empty strings as unknown.
type CloudEventHeaders struct {
	SpecVersion string
	Type        string
	Source      string
	ID          string
	Time        string
}

// ExtractCloudEventHeaders reads CloudEvents Binary Mode headers from a franz-go record.
func ExtractCloudEventHeaders(headers []kgo.RecordHeader) CloudEventHeaders {
	var ce CloudEventHeaders
	for _, h := range headers {
		switch h.Key {
		case "ce-specversion":
			ce.SpecVersion = string(h.Value)
		case "ce-type":
			ce.Type = string(h.Value)
		case "ce-source":
			ce.Source = string(h.Value)
		case "ce-id":
			ce.ID = string(h.Value)
		case "ce-time":
			ce.Time = string(h.Value)
		}
	}
	return ce
}
