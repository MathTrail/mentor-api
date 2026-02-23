package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerPort string `mapstructure:"SERVER_PORT"`

	// Dapr
	DaprHost       string `mapstructure:"DAPR_HOST"`
	DaprPort       string `mapstructure:"DAPR_PORT"`
	DBBindingName  string `mapstructure:"DB_BINDING_NAME"` // Dapr binding component name for DB access (server binary)
	DaprMaxRetries int    `mapstructure:"DAPR_MAX_RETRIES"`

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
	v.SetDefault("DAPR_HOST", "localhost")
	v.SetDefault("DAPR_PORT", "3500")
	v.SetDefault("DB_BINDING_NAME", "mentor-db")
	v.SetDefault("DAPR_MAX_RETRIES", 10)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("APP_NAME", "mentor-api")
	v.SetDefault("OTEL_ENDPOINT", "")
	v.SetDefault("OTEL_SAMPLE_RATE", 0.1)
	v.SetDefault("PYROSCOPE_ENDPOINT", "")
	v.SetDefault("LLM_TIMEOUT", "10s")
	v.SetDefault("SHUTDOWN_TIMEOUT", "5s")

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

	return cfg
}
