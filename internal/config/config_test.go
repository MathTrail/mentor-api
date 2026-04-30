package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

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
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LLM_TIMEOUT", "30s")
	t.Setenv("SHUTDOWN_TIMEOUT", "15s")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

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

func TestLoadInvalidLLMTimeout(t *testing.T) {
	t.Setenv("LLM_TIMEOUT", "not-a-duration")
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LLM_TIMEOUT, got nil")
	}
}

func TestLoadInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "bad")
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid SHUTDOWN_TIMEOUT, got nil")
	}
}

func TestLoadOTelSampleRateTooHigh(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "1.5")
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for OTEL_SAMPLE_RATE > 1, got nil")
	}
}

func TestLoadOTelSampleRateNegative(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "-0.1")
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative OTEL_SAMPLE_RATE, got nil")
	}
}

func TestLoadOTelSampleRateValidBoundaries(t *testing.T) {
	t.Setenv("PG_CREDENTIALS_DIR", t.TempDir())
	t.Setenv("OTEL_SAMPLE_RATE", "0.0")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OTelSampleRate != 0.0 {
		t.Errorf("OTelSampleRate: got %v, want 0.0", cfg.OTelSampleRate)
	}

	t.Setenv("OTEL_SAMPLE_RATE", "1.0")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OTelSampleRate != 1.0 {
		t.Errorf("OTelSampleRate: got %v, want 1.0", cfg.OTelSampleRate)
	}
}
func TestPostgresDSN(t *testing.T) {
	cfg := &Config{
		PgHost:     "localhost",
		PgPort:     "5432",
		PgDatabase: "testdb",
		PgSSLMode:  "disable",
	}
	want := "host=localhost port=5432 dbname=testdb sslmode=disable"
	if got := cfg.PostgresDSN(); got != want {
		t.Errorf("PostgresDSN() = %q, want %q", got, want)
	}
}

func TestValidate_OTelSampleRateTooHigh(t *testing.T) {
	cfg := &Config{
		OTelSampleRate:   1.5,
		PgCredentialsDir: t.TempDir(),
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for OTelSampleRate > 1, got nil")
	}
}

func TestValidate_EmptyPgCredentialsDir(t *testing.T) {
	cfg := &Config{
		OTelSampleRate:   0.5,
		PgCredentialsDir: "",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty PgCredentialsDir, got nil")
	}
}
