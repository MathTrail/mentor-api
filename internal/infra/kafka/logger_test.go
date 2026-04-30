package kafka

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

func TestZapKafkaLogger_Level(t *testing.T) {
	l := newZapLogger(zap.NewNop())
	if l.Level() != kgo.LogLevelDebug {
		t.Errorf("Level() = %v, want LogLevelDebug", l.Level())
	}
}

func TestZapKafkaLogger_AllLevels(t *testing.T) {
	l := newZapLogger(zap.NewNop())
	levels := []kgo.LogLevel{
		kgo.LogLevelError,
		kgo.LogLevelWarn,
		kgo.LogLevelInfo,
		kgo.LogLevelDebug,
	}
	for _, lvl := range levels {
		// Should not panic.
		l.Log(lvl, "test message", "key", "value")
	}
}

func TestZapKafkaLogger_OddKeyvals(t *testing.T) {
	l := newZapLogger(zap.NewNop())
	// Odd number of keyvals — trailing key has no value; must not panic.
	l.Log(kgo.LogLevelInfo, "odd keyvals", "orphan-key")
}

func TestZapKafkaLogger_NonStringKey(t *testing.T) {
	l := newZapLogger(zap.NewNop())
	// Non-string key should be silently ignored, not panic.
	l.Log(kgo.LogLevelInfo, "bad key", 42, "value")
}
