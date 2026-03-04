package config

import (
	"testing"
	"time"
)

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic, got none")
		}
	}()
	fn()
}

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort: got %q, want %q", cfg.ServerPort, "8080")
	}
	if !cfg.SwaggerEnabled {
		t.Error("SwaggerEnabled: want true by default")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat: got %q, want %q", cfg.LogFormat, "json")
	}
	if cfg.LLMTimeout != 10*time.Second {
		t.Errorf("LLMTimeout: got %v, want %v", cfg.LLMTimeout, 10*time.Second)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("ShutdownTimeout: got %v, want %v", cfg.ShutdownTimeout, 5*time.Second)
	}
	if cfg.OTelSampleRate != 0.1 {
		t.Errorf("OTelSampleRate: got %v, want %v", cfg.OTelSampleRate, 0.1)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LLM_TIMEOUT", "30s")
	t.Setenv("SHUTDOWN_TIMEOUT", "15s")

	cfg := Load()

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort: got %q, want %q", cfg.ServerPort, "9090")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LLMTimeout != 30*time.Second {
		t.Errorf("LLMTimeout: got %v, want %v", cfg.LLMTimeout, 30*time.Second)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout: got %v, want %v", cfg.ShutdownTimeout, 15*time.Second)
	}
}

func TestLoadInvalidLLMTimeoutPanics(t *testing.T) {
	t.Setenv("LLM_TIMEOUT", "not-a-duration")
	assertPanics(t, func() { Load() })
}

func TestLoadInvalidShutdownTimeoutPanics(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "bad")
	assertPanics(t, func() { Load() })
}

func TestLoadOTelSampleRateTooHighPanics(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "1.5")
	assertPanics(t, func() { Load() })
}

func TestLoadOTelSampleRateNegativePanics(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "-0.1")
	assertPanics(t, func() { Load() })
}

func TestLoadOTelSampleRateValidBoundaries(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "0.0")
	cfg := Load()
	if cfg.OTelSampleRate != 0.0 {
		t.Errorf("OTelSampleRate: got %v, want 0.0", cfg.OTelSampleRate)
	}

	t.Setenv("OTEL_SAMPLE_RATE", "1.0")
	cfg = Load()
	if cfg.OTelSampleRate != 1.0 {
		t.Errorf("OTelSampleRate: got %v, want 1.0", cfg.OTelSampleRate)
	}
}
