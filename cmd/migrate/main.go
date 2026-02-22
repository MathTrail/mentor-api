package main

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/logging"
	"github.com/MathTrail/mentor-api/migrations"
)

func main() {
	logger := logging.NewLogger("info")

	cfg := config.Load()

	// Ensure the target database exists (Goose requires it).
	ensureDatabase(cfg, logger)

	// Run Goose migrations with embedded SQL files.
	if err := runMigrations(cfg.DSN(), logger); err != nil {
		logger.Fatal("migrations failed", zap.Error(err))
	}

	logger.Info("all migrations completed successfully")
	fmt.Println("✓ Database migrations completed successfully")
}

// runMigrations applies all pending Goose migrations from the embedded FS.
func runMigrations(dsn string, logger *zap.Logger) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
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
func ensureDatabase(cfg *config.Config, logger *zap.Logger) {
	adminDSN := cfg.DSNForDB("postgres")

	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		logger.Fatal("failed to connect to admin database", zap.Error(err))
	}
	defer func() { _ = db.Close() }()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists)
	if err != nil {
		logger.Fatal("failed to check database existence", zap.Error(err))
	}

	if !exists {
		// CREATE DATABASE cannot run inside a transaction.
		if _, err := db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName)); err != nil {
			logger.Fatal("failed to create database", zap.String("dbname", cfg.DBName), zap.Error(err))
		}
		logger.Info("database created", zap.String("dbname", cfg.DBName))
	} else {
		logger.Info("database already exists", zap.String("dbname", cfg.DBName))
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
