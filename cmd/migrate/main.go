package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/logging"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger("info")

	// Load configuration
	cfg := config.Load()

	// Ensure the target database exists
	ensureDatabase(cfg, logger)

	// Run SQL migrations (extensions, custom types, tables)
	runSQLMigrations(cfg.DSN(), logger)

	logger.Info("all migrations completed successfully")
	fmt.Println("✓ Database migrations completed successfully")
}

// runSQLMigrations executes all .sql files from the migrations directory.
func runSQLMigrations(dsn string, logger *zap.Logger) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Fatal("failed to open sql connection for migrations", zap.Error(err))
	}
	defer func() { _ = sqlDB.Close() }()

	migrationsDir := "/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "./migrations"
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		logger.Fatal("failed to read migrations directory", zap.Error(err))
	}

	if len(files) == 0 {
		logger.Info("no SQL migration files found", zap.String("directory", migrationsDir))
		return
	}

	sort.Strings(files)
	logger.Info("running SQL migrations", zap.Int("count", len(files)))

	for _, file := range files {
		logger.Info("executing migration", zap.String("file", filepath.Base(file)))

		content, err := os.ReadFile(file)
		if err != nil {
			logger.Fatal("failed to read migration file", zap.String("file", file), zap.Error(err))
		}

		if _, err := sqlDB.Exec(string(content)); err != nil {
			logger.Fatal("migration failed",
				zap.String("file", filepath.Base(file)),
				zap.Error(err),
			)
		}

		logger.Info("migration completed", zap.String("file", filepath.Base(file)))
	}
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
		// CREATE DATABASE cannot run inside a transaction
		if _, err := db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName)); err != nil {
			logger.Fatal("failed to create database", zap.String("dbname", cfg.DBName), zap.Error(err))
		}
		logger.Info("database created", zap.String("dbname", cfg.DBName))
	} else {
		logger.Info("database already exists", zap.String("dbname", cfg.DBName))
	}
}
