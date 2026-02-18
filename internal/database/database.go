package database

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/MathTrail/mentor-api/internal/config"
)

// NewPool creates a pgx connection pool using the provided config.
func NewPool(ctx context.Context, cfg *config.Config, logger *zap.Logger) *pgxpool.Pool {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&default_query_exec_mode=simple_protocol",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Fatal("failed to parse database config", zap.Error(err))
	}

	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()

	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Fatal("failed to create connection pool", zap.Error(err))
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("failed to ping database", zap.Error(err))
	}

	logger.Info("database connection pool established",
		zap.String("host", cfg.DBHost),
		zap.String("port", cfg.DBPort),
		zap.String("dbname", cfg.DBName),
	)

	return pool
}
