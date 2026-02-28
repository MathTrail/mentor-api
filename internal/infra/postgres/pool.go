package postgres

import (
	"context"
	"fmt"

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
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
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
	defer rows.Close()

	fieldDescs := rows.FieldDescriptions()
	var result []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("pgxpool scan row: %w", err)
		}
		row := make(map[string]any, len(fieldDescs))
		for i, fd := range fieldDescs {
			row[fd.Name] = values[i]
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgxpool rows: %w", err)
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
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
