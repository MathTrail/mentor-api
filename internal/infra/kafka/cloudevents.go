package kafka

import "github.com/twmb/franz-go/pkg/kgo"

// CloudEventHeaders holds the CloudEvents Binary Mode attributes extracted from
// Kafka record headers. Attributes are written by RisingWave sink via the
// _headers MAP column.
//
// Missing headers produce empty strings — older records without CE attributes are
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
		case "ce_specversion":
			ce.SpecVersion = string(h.Value)
		case "ce_type":
			ce.Type = string(h.Value)
		case "ce_source":
			ce.Source = string(h.Value)
		case "ce_id":
			ce.ID = string(h.Value)
		case "ce_time":
			ce.Time = string(h.Value)
		}
	}
	return ce
}
