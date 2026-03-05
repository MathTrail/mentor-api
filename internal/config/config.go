package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerPort     string `mapstructure:"SERVER_PORT"`
	SwaggerEnabled bool   `mapstructure:"SWAGGER_ENABLED"`

	// PostgreSQL connection parameters (non-sensitive).
	// Credentials are read from mounted Secret files; see PgCredentialsDir.
	PgHost           string `mapstructure:"PG_HOST"`
	PgPort           string `mapstructure:"PG_PORT"`
	PgDatabase       string `mapstructure:"PG_DATABASE"`
	PgSSLMode        string `mapstructure:"PG_SSL_MODE"`
	PgCredentialsDir string `mapstructure:"PG_CREDENTIALS_DIR"` // directory with "username" and "password" files (VSO volume mount)

	// Logging
	LogLevel  string `mapstructure:"LOG_LEVEL"`
	LogFormat string `mapstructure:"LOG_FORMAT"` // "json" (default) or "console" (colored dev output)

	// Observability
	ServiceName       string  `mapstructure:"APP_NAME"`
	OTelEndpoint      string  `mapstructure:"OTEL_ENDPOINT"`
	OTelSampleRate    float64 `mapstructure:"OTEL_SAMPLE_RATE"` // 0.0–1.0; fraction of root spans sampled
	PyroscopeEndpoint string  `mapstructure:"PYROSCOPE_ENDPOINT"`

	// Timeouts
	LLMTimeoutRaw      string        `mapstructure:"LLM_TIMEOUT"` // e.g. "10s"
	LLMTimeout         time.Duration // parsed from LLMTimeoutRaw in Load()
	ShutdownTimeoutRaw string        `mapstructure:"SHUTDOWN_TIMEOUT"` // e.g. "5s", "10s"
	ShutdownTimeout    time.Duration // parsed from ShutdownTimeoutRaw in Load()
}

func Load() *Config {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("SWAGGER_ENABLED", true)
	v.SetDefault("PG_HOST", "postgres-pgbouncer")
	v.SetDefault("PG_PORT", "6432")
	v.SetDefault("PG_DATABASE", "mentor")
	v.SetDefault("PG_SSL_MODE", "disable")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("APP_NAME", "mentor-api")
	v.SetDefault("OTEL_ENDPOINT", "")
	v.SetDefault("OTEL_SAMPLE_RATE", 0.1)
	v.SetDefault("PYROSCOPE_ENDPOINT", "")
	v.SetDefault("LLM_TIMEOUT", "10s")
	v.SetDefault("SHUTDOWN_TIMEOUT", "5s")
	v.SetDefault("PG_CREDENTIALS_DIR", "")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config: %v", err))
	}

	// Parse durations from strings.
	// Kept separate because Viper cannot unmarshal time.Duration from env vars.
	llmD, err := time.ParseDuration(cfg.LLMTimeoutRaw)
	if err != nil {
		panic(fmt.Sprintf("invalid LLM_TIMEOUT %q: %v", cfg.LLMTimeoutRaw, err))
	}
	cfg.LLMTimeout = llmD

	shutD, err := time.ParseDuration(cfg.ShutdownTimeoutRaw)
	if err != nil {
		panic(fmt.Sprintf("invalid SHUTDOWN_TIMEOUT %q: %v", cfg.ShutdownTimeoutRaw, err))
	}
	cfg.ShutdownTimeout = shutD

	if cfg.OTelSampleRate < 0 || cfg.OTelSampleRate > 1 {
		panic(fmt.Sprintf("OTEL_SAMPLE_RATE must be between 0.0 and 1.0, got %v", cfg.OTelSampleRate))
	}

	if cfg.PgCredentialsDir == "" {
		panic("PG_CREDENTIALS_DIR is required: set it to the directory containing the VSO-mounted username and password files")
	}

	return cfg
}

// PostgresDSN returns the non-sensitive part of the PostgreSQL connection string.
// Credentials are managed separately via PgCredentialsDir.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		c.PgHost, c.PgPort, c.PgDatabase, c.PgSSLMode,
	)
}
