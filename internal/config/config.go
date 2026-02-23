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
	LogLevel string `mapstructure:"LOG_LEVEL"`

	// Observability
	ServiceName       string `mapstructure:"APP_NAME"`
	OTelEndpoint      string `mapstructure:"OTEL_ENDPOINT"`
	PyroscopeEndpoint string `mapstructure:"PYROSCOPE_ENDPOINT"`

	// Lifecycle
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
	v.SetDefault("APP_NAME", "mentor-api")
	v.SetDefault("OTEL_ENDPOINT", "")
	v.SetDefault("PYROSCOPE_ENDPOINT", "")
	v.SetDefault("SHUTDOWN_TIMEOUT", "5s")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config: %v", err))
	}

	// Parse shutdown timeout from string to time.Duration.
	// Kept separate because Viper cannot unmarshal time.Duration from env vars.
	d, err := time.ParseDuration(cfg.ShutdownTimeoutRaw)
	if err != nil {
		panic(fmt.Sprintf("invalid SHUTDOWN_TIMEOUT %q: %v", cfg.ShutdownTimeoutRaw, err))
	}
	cfg.ShutdownTimeout = d

	return cfg
}
