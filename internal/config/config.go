package config

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	defaultServerPort              = "8080"
	defaultPgHost                  = "postgres-pgbouncer"
	defaultPgPort                  = "6432"
	defaultPgDatabase              = "mentor"
	defaultPgSSLMode               = "disable"
	defaultLogLevel                = "info"
	defaultLogFormat               = "json"
	defaultAppName                 = "mentor-api"
	defaultOTelSampleRate          = 0.1
	defaultLLMTimeout              = "10s"
	defaultShutdownTimeout         = "5s"
	defaultReadHeaderTimeout       = "5s"
	defaultReadTimeout             = "5s"
	defaultWriteTimeout            = "10s"
	defaultIdleTimeout             = "120s"
	defaultKafkaBootstrapServers   = "automq:9092"
	defaultKafkaConsumerGroup      = "mentor-api"
	defaultPodName                 = "mentor-api-local"
	defaultApicurioURL             = "http://mathtrail-apicurio-apicurio-registry:8080/apis/ccompat/v7"
	defaultOnboardingSchemaSubject = "students.v1.StudentOnboardingReady"
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

	// Kafka (students.onboarding.ready consumer)
	KafkaBootstrapServers string `mapstructure:"KAFKA_BOOTSTRAP_SERVERS"`
	KafkaConsumerGroup    string `mapstructure:"KAFKA_CONSUMER_GROUP"`
	KafkaSASLUsername     string `mapstructure:"KAFKA_SASL_USERNAME"`
	KafkaSASLPassword     string `mapstructure:"KAFKA_SASL_PASSWORD"`
	// InstanceID is the static Kafka consumer group member ID.
	// Injected as POD_NAME by Kubernetes; prevents rebalance storms during rolling deploys.
	InstanceID string `mapstructure:"POD_NAME"`

	// ApicurioURL is the base URL of the Confluent-compat v7 API, used by TopicValidator
	// to fetch the latest schema ID at startup (fail-fast if unreachable).
	ApicurioURL             string `mapstructure:"APICURIO_URL"`
	OnboardingSchemaSubject string `mapstructure:"ONBOARDING_SCHEMA_SUBJECT"`

	// Timeouts
	LLMTimeout        time.Duration `mapstructure:"LLM_TIMEOUT"`
	ShutdownTimeout   time.Duration `mapstructure:"SHUTDOWN_TIMEOUT"`
	ReadHeaderTimeout time.Duration `mapstructure:"HTTP_READ_HEADER_TIMEOUT"`
	ReadTimeout       time.Duration `mapstructure:"HTTP_READ_TIMEOUT"`
	WriteTimeout      time.Duration `mapstructure:"HTTP_WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `mapstructure:"HTTP_IDLE_TIMEOUT"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("SERVER_PORT", defaultServerPort)
	v.SetDefault("SWAGGER_ENABLED", true)
	v.SetDefault("PG_HOST", defaultPgHost)
	v.SetDefault("PG_PORT", defaultPgPort)
	v.SetDefault("PG_DATABASE", defaultPgDatabase)
	v.SetDefault("PG_SSL_MODE", defaultPgSSLMode)
	v.SetDefault("LOG_LEVEL", defaultLogLevel)
	v.SetDefault("LOG_FORMAT", defaultLogFormat)
	v.SetDefault("APP_NAME", defaultAppName)
	v.SetDefault("OTEL_ENDPOINT", "")
	v.SetDefault("OTEL_SAMPLE_RATE", defaultOTelSampleRate)
	v.SetDefault("PYROSCOPE_ENDPOINT", "")
	v.SetDefault("LLM_TIMEOUT", defaultLLMTimeout)
	v.SetDefault("SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	v.SetDefault("HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout)
	v.SetDefault("HTTP_READ_TIMEOUT", defaultReadTimeout)
	v.SetDefault("HTTP_WRITE_TIMEOUT", defaultWriteTimeout)
	v.SetDefault("HTTP_IDLE_TIMEOUT", defaultIdleTimeout)
	v.SetDefault("KAFKA_BOOTSTRAP_SERVERS", defaultKafkaBootstrapServers)
	v.SetDefault("KAFKA_CONSUMER_GROUP", defaultKafkaConsumerGroup)
	v.SetDefault("POD_NAME", defaultPodName)
	v.SetDefault("APICURIO_URL", defaultApicurioURL)
	v.SetDefault("ONBOARDING_SCHEMA_SUBJECT", defaultOnboardingSchemaSubject)
	v.SetDefault("PG_CREDENTIALS_DIR", "")

	cfg := &Config{}
	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))
	if err := v.Unmarshal(cfg, decodeHook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that config values are within acceptable bounds.
func (c *Config) Validate() error {
	if c.OTelSampleRate < 0 || c.OTelSampleRate > 1 {
		return fmt.Errorf("OTEL_SAMPLE_RATE must be between 0.0 and 1.0, got %v", c.OTelSampleRate)
	}
	if c.PgCredentialsDir == "" {
		return fmt.Errorf("PG_CREDENTIALS_DIR is required: set it to the directory containing the VSO-mounted username and password files")
	}
	return nil
}

// PostgresDSN returns the non-sensitive part of the PostgreSQL connection string.
// Credentials are managed separately via PgCredentialsDir.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		c.PgHost, c.PgPort, c.PgDatabase, c.PgSSLMode,
	)
}
