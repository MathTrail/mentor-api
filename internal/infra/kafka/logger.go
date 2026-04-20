package kafka

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type zapKafkaLogger struct{ l *zap.Logger }

// newZapLogger wraps a zap.Logger as a kgo.Logger.
// Level always returns LogLevelDebug so that kgo passes all messages through;
// zap's own configured level then applies the final filtering.
func newZapLogger(l *zap.Logger) kgo.Logger {
	return &zapKafkaLogger{l: l.Named("kafka")}
}

func (z *zapKafkaLogger) Level() kgo.LogLevel { return kgo.LogLevelDebug }

func (z *zapKafkaLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields = append(fields, zap.Any(key, keyvals[i+1]))
		}
	}
	switch level {
	case kgo.LogLevelError:
		z.l.Error(msg, fields...)
	case kgo.LogLevelWarn:
		z.l.Warn(msg, fields...)
	case kgo.LogLevelInfo:
		z.l.Info(msg, fields...)
	default:
		z.l.Debug(msg, fields...)
	}
}

// compile-time interface check
var _ kgo.Logger = (*zapKafkaLogger)(nil)
