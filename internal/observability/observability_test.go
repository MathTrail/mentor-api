package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/MathTrail/mentor-api/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func testConfig() *config.Config {
	return &config.Config{
		ServiceName:       "test-service",
		OTelEndpoint:      "", // disabled — no external connection required
		OTelSampleRate:    0.1,
		PyroscopeEndpoint: "", // disabled
	}
}

func TestNewNotNil(t *testing.T) {
	o := New(testConfig(), zap.NewNop())
	if o == nil {
		t.Error("New returned nil")
	}
}

func TestNewStoresCfgAndLogger(t *testing.T) {
	cfg := testConfig()
	o := New(cfg, zap.NewNop())
	if o.cfg != cfg {
		t.Error("cfg not stored correctly")
	}
}

func TestShutdownNothingInitialized(t *testing.T) {
	o := New(testConfig(), zap.NewNop())
	// Must not panic when no components have been initialised.
	o.Shutdown(context.Background())
}

func TestShutdownCallsTracerShutdown(t *testing.T) {
	called := false
	o := New(testConfig(), zap.NewNop())
	o.tracerShutdown = func(_ context.Context) error {
		called = true
		return nil
	}
	o.Shutdown(context.Background())
	if !called {
		t.Error("tracerShutdown was not called")
	}
}

func TestShutdownCallsMetricsShutdown(t *testing.T) {
	called := false
	o := New(testConfig(), zap.NewNop())
	o.metricsShutdown = func(_ context.Context) error {
		called = true
		return nil
	}
	o.Shutdown(context.Background())
	if !called {
		t.Error("metricsShutdown was not called")
	}
}

func TestShutdownLogsTracerError(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(core)

	o := New(testConfig(), logger)
	o.tracerShutdown = func(_ context.Context) error {
		return errors.New("tracer boom")
	}
	o.Shutdown(context.Background())

	entries := logs.FilterMessage("tracer shutdown error").All()
	if len(entries) == 0 {
		t.Error("expected warn log for tracer shutdown error")
	}
}

func TestShutdownLogsMetricsError(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(core)

	o := New(testConfig(), logger)
	o.metricsShutdown = func(_ context.Context) error {
		return errors.New("metrics boom")
	}
	o.Shutdown(context.Background())

	entries := logs.FilterMessage("metrics shutdown error").All()
	if len(entries) == 0 {
		t.Error("expected warn log for metrics shutdown error")
	}
}

func TestShutdownCallsProfilerStop(t *testing.T) {
	called := false
	o := New(testConfig(), zap.NewNop())
	o.profiler = &stubProfiler{stopFn: func() error {
		called = true
		return nil
	}}
	o.Shutdown(context.Background())
	if !called {
		t.Error("profiler.Stop was not called")
	}
}

// TestInitNoExternalEndpoints verifies that Init succeeds when no external
// observability backends are configured (only the in-process Prometheus
// exporter is started). This test is intentionally run once per binary to
// avoid duplicate Prometheus metric registration panics.
func TestInitNoExternalEndpoints(t *testing.T) {
	o := New(testConfig(), zap.NewNop())
	if err := o.Init(context.Background()); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	// Tracer and profiler should be nil since endpoints are empty.
	if o.tracerShutdown != nil {
		t.Error("tracerShutdown should be nil when OTelEndpoint is empty")
	}
	if o.profiler != nil {
		t.Error("profiler should be nil when PyroscopeEndpoint is empty")
	}
	if o.metricsShutdown == nil {
		t.Error("metricsShutdown should be set (Prometheus exporter always starts)")
	}
	// Graceful cleanup.
	o.Shutdown(context.Background())
}

// stubProfiler is a minimal implementation of the profiler interface used in Observability.
type stubProfiler struct {
	stopFn func() error
}

func (s *stubProfiler) Stop() error {
	if s.stopFn != nil {
		return s.stopFn()
	}
	return nil
}
