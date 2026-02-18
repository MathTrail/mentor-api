package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerPort string `mapstructure:"SERVER_PORT"`

	// Database
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`

	// Dapr
	DaprHost      string `mapstructure:"DAPR_HOST"`
	DaprPort      string `mapstructure:"DAPR_PORT"`
	DBBindingName string `mapstructure:"DB_BINDING_NAME"` // Dapr binding component name for DB access (server binary)

	// Logging
	LogLevel string `mapstructure:"LOG_LEVEL"`

	// Observability
	ServiceName       string `mapstructure:"APP_NAME"`
	OTelEndpoint      string `mapstructure:"OTEL_ENDPOINT"`
	PyroscopeEndpoint string `mapstructure:"PYROSCOPE_ENDPOINT"`
}

func Load() *Config {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("DB_HOST", "postgres-pgbouncer")
	v.SetDefault("DB_PORT", "6432")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "mentor")
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("DAPR_HOST", "localhost")
	v.SetDefault("DAPR_PORT", "3500")
	v.SetDefault("DB_BINDING_NAME", "mentor-db")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("APP_NAME", "mentor-api")
	v.SetDefault("OTEL_ENDPOINT", "otel-collector-opentelemetry-collector.monitoring.svc.cluster.local:4317")
	v.SetDefault("PYROSCOPE_ENDPOINT", "http://pyroscope.monitoring.svc.cluster.local:4040")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config: %v", err))
	}

	return cfg
}

func (c *Config) DSN() string {
	return c.DSNForDB(c.DBName)
}

// DSNForDB returns a connection string targeting the given database.
func (c *Config) DSNForDB(dbname string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, dbname, c.DBSSLMode,
	)
}
