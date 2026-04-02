package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql used by Goose
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/MathTrail/mentor-api/internal/logger"
	"github.com/MathTrail/mentor-api/migrations"
)

// dbConfig holds database connection parameters read from environment variables.
// These variables are injected only into the migration K8s Job, keeping
// credentials out of the server binary's environment.
type dbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func loadDBConfig() (dbConfig, error) {
	host, err := requiredEnv("DB_HOST")
	if err != nil {
		return dbConfig{}, err
	}
	port, err := requiredEnv("DB_PORT")
	if err != nil {
		return dbConfig{}, err
	}
	user, err := requiredEnv("DB_USER")
	if err != nil {
		return dbConfig{}, err
	}
	password, err := requiredEnv("DB_PASSWORD")
	if err != nil {
		return dbConfig{}, err
	}
	name, err := requiredEnv("DB_NAME")
	if err != nil {
		return dbConfig{}, err
	}
	return dbConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Name:     name,
		SSLMode:  envOrDefault("DB_SSL_MODE", "disable"),
	}, nil
}

// requiredEnv reads an environment variable or returns an error if it is empty.
// A missing variable means the K8s Job is misconfigured — fail fast.
func requiredEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dsn builds a PostgreSQL URI for the given database name.
// User and password are URL-encoded to handle special characters safely.
func (c dbConfig) dsn(dbname string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		c.Host, c.Port, dbname, c.SSLMode,
	)
}

var dbNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateDBName(name string) error {
	if !dbNameRe.MatchString(name) {
		return fmt.Errorf("invalid database name %q: must match ^[a-zA-Z0-9_-]+$", name)
	}
	return nil
}

func main() {
	logger := logger.NewLogger("mentor-migrate", "info", "json")

	cfg, err := loadDBConfig()
	if err != nil {
		logger.Fatal("missing required config", zap.Error(err))
	}

	// Ensure the target database exists (Goose requires it).
	ensureDatabase(cfg, logger)

	// Run Goose migrations with embedded SQL files.
	if err := runMigrations(cfg.dsn(cfg.Name), logger); err != nil {
		logger.Fatal("migrations failed", zap.Error(err))
	}

	logger.Info("all migrations completed successfully")
}

// runMigrations applies all pending Goose migrations from the embedded FS.
func runMigrations(dsn string, logger *zap.Logger) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	// Goose applies migrations sequentially.
	// Limit the pool to one connection for predictable migration execution.
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(&gooseLogAdapter{l: logger})

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	logger.Info("starting goose migrations")
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up failed: %w", err)
	}

	return nil
}

// ensureDatabase connects to the default "postgres" database and creates the
// target database if it does not already exist.
func ensureDatabase(cfg dbConfig, logger *zap.Logger) {
	if err := validateDBName(cfg.Name); err != nil {
		logger.Fatal("invalid database name", zap.Error(err))
	}

	adminDSN := cfg.dsn("postgres")

	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		logger.Fatal("failed to connect to admin database", zap.Error(err))
	}
	defer func() { _ = db.Close() }()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.Name).Scan(&exists)
	if err != nil {
		logger.Fatal("failed to check database existence", zap.Error(err))
	}

	if !exists {
		// CREATE DATABASE cannot run inside a transaction.
		if _, err := db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.Name)); err != nil {
			logger.Fatal("failed to create database", zap.String("dbname", cfg.Name), zap.Error(err))
		}
		logger.Info("database created", zap.String("dbname", cfg.Name))
	} else {
		logger.Info("database already exists", zap.String("dbname", cfg.Name))
	}
}

// gooseLogAdapter bridges Goose's logger interface to zap.
type gooseLogAdapter struct {
	l *zap.Logger
}

func (a *gooseLogAdapter) Printf(format string, v ...interface{}) {
	a.l.Info(fmt.Sprintf(format, v...))
}

func (a *gooseLogAdapter) Fatalf(format string, v ...interface{}) {
	a.l.Fatal(fmt.Sprintf(format, v...))
}
