package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvPgPool implements DB using a pgxpool.Pool whose credentials are
// provided via environment variables (injected by VSO from the
// mentor-api-db-secret K8s Secret). Credential rotation is handled
// externally: VSO triggers a rolling restart of the Deployment when it
// renews the Vault dynamic secret lease.
type EnvPgPool struct {
	pool *pgxpool.Pool
}

// NewEnvPgPool opens a connection pool using the supplied DSN.
func NewEnvPgPool(ctx context.Context, dsn string) (*EnvPgPool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}
	return &EnvPgPool{pool: pool}, nil
}

// Query executes a SQL statement and returns result rows as a slice of maps.
// Column names are taken from the pgx FieldDescriptions.
func (p *EnvPgPool) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	rows, err := p.pool.Query(ctx, sql, params...)
	if err != nil {
		return nil, fmt.Errorf("pgxpool query: %w", err)
	}
	return scanRows(rows)
}

// Exec executes a SQL statement that produces no result rows.
func (p *EnvPgPool) Exec(ctx context.Context, sql string, params ...any) error {
	_, err := p.pool.Exec(ctx, sql, params...)
	if err != nil {
		return fmt.Errorf("pgxpool exec: %w", err)
	}
	return nil
}

// Ping verifies the connection pool is reachable.
func (p *EnvPgPool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}
