package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MathTrail/mentor-api/internal/config"
	"github.com/MathTrail/mentor-api/internal/database"
	"github.com/MathTrail/mentor-api/internal/logging"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger("info")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db := database.NewConnection(cfg, logger)

	// Get underlying sql.DB for raw queries
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to get sql.DB", zap.Error(err))
	}

	// Read migration files
	migrationsDir := "/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "./migrations"
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		logger.Fatal("failed to read migrations directory", zap.Error(err))
	}

	if len(files) == 0 {
		logger.Warn("no migration files found", zap.String("directory", migrationsDir))
		return
	}

	// Sort files by name (assumes numeric prefix like 001_init.sql)
	sort.Strings(files)

	logger.Info("running migrations", zap.Int("count", len(files)))

	// Execute each migration file
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

	logger.Info("all migrations completed successfully")
	fmt.Println("✓ Database migrations completed successfully")
}
