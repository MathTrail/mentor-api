package postgres

import "context"

// DB abstracts data access so that the application can swap between
// a Vault-managed pool (production) and a direct pgx pool (tests) transparently.
type DB interface {
	// Query executes a SQL statement and returns the result rows as a slice of maps.
	Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error)

	// Exec executes a SQL statement that produces no result rows.
	Exec(ctx context.Context, sql string, params ...any) error

	// Ping verifies the underlying connection is reachable.
	Ping(ctx context.Context) error
}
