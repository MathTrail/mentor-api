package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DaprPgPool implements DB using a pgxpool.Pool whose credentials are fetched
// from a Dapr secret store (e.g. the "vault-db" component backed by Vault's
// Database Secrets Engine).
//
// On startup, NewDaprPgPool fetches a dynamic credential lease via Dapr
// (creating username + password) and opens a connection pool. A background
// goroutine created by StartRefresh replaces the pool before the credential
// TTL expires, swapping pools atomically under a read-write lock so in-flight
// queries are never interrupted.
//
// Each GetSecret call to the secret store creates a NEW credential lease.
// The old lease (and its DB user) is revoked by the backend after the TTL;
// the 30-second grace period in the refresh goroutine ensures all queries
// against the old pool finish before it is closed.
type DaprPgPool struct {
	mu          sync.RWMutex
	pool        *pgxpool.Pool
	daprClient  dapr.Client
	secretStore string // Dapr secret store component name, e.g. "vault-db"
	secretKey   string // Vault path, e.g. "creds/mentor-api-role"
	pgDSNTpl    string // connection string without user/password, e.g. "host=... dbname=... sslmode=disable"
	logger      *zap.Logger
}

// NewDaprPgPool creates a DaprPgPool and initialises the first connection pool.
// pgDSNTpl must be a partial libpq connection string containing host, port,
// dbname, and sslmode — but NOT user or password (those come from Vault).
func NewDaprPgPool(
	ctx context.Context,
	daprClient dapr.Client,
	secretStore, secretKey, pgDSNTpl string,
	logger *zap.Logger,
) (*DaprPgPool, error) {
	p := &DaprPgPool{
		daprClient:  daprClient,
		secretStore: secretStore,
		secretKey:   secretKey,
		pgDSNTpl:    pgDSNTpl,
		logger:      logger,
	}
	if err := p.initPool(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

// StartRefresh runs a background goroutine that refreshes credentials every
// interval. interval should be shorter than the Vault default_ttl
// (e.g. 50m for a 1h TTL). The goroutine stops when ctx is cancelled.
func (p *DaprPgPool) StartRefresh(ctx context.Context, interval time.Duration) {
	go p.startRefresh(ctx, interval)
}

func (p *DaprPgPool) startRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			username, password, err := p.fetchCreds(ctx)
			if err != nil {
				p.logger.Error("failed to refresh vault db credentials", zap.Error(err))
				continue
			}

			dsn := fmt.Sprintf("%s user=%s password=%s", p.pgDSNTpl, username, password)
			newPool, err := pgxpool.New(ctx, dsn)
			if err != nil {
				p.logger.Error("failed to create new pgxpool after credential refresh", zap.Error(err))
				continue
			}

			p.mu.Lock()
			oldPool := p.pool
			p.pool = newPool
			p.mu.Unlock()

			// Allow in-flight queries on the old pool to finish before closing.
			// The 30-second grace period covers normal query latency.
			go func(op *pgxpool.Pool) {
				time.Sleep(30 * time.Second)
				op.Close()
			}(oldPool)

			p.logger.Info("successfully rotated vault db credentials",
				zap.String("secretStore", p.secretStore),
				zap.String("secretKey", p.secretKey),
			)
		}
	}
}

// Query executes a SQL statement and returns result rows as a slice of maps.
// Column names are taken from the pgx FieldDescriptions.
func (p *DaprPgPool) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	p.mu.RLock()
	pool := p.pool
	p.mu.RUnlock()

	rows, err := pool.Query(ctx, sql, params...)
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
func (p *DaprPgPool) Exec(ctx context.Context, sql string, params ...any) error {
	p.mu.RLock()
	pool := p.pool
	p.mu.RUnlock()

	_, err := pool.Exec(ctx, sql, params...)
	if err != nil {
		return fmt.Errorf("pgxpool exec: %w", err)
	}
	return nil
}

// Ping verifies the connection pool is reachable.
func (p *DaprPgPool) Ping(ctx context.Context) error {
	p.mu.RLock()
	pool := p.pool
	p.mu.RUnlock()

	return pool.Ping(ctx)
}

// initPool fetches credentials from Vault and opens the first connection pool.
func (p *DaprPgPool) initPool(ctx context.Context) error {
	username, password, err := p.fetchCreds(ctx)
	if err != nil {
		return err
	}
	dsn := fmt.Sprintf("%s user=%s password=%s", p.pgDSNTpl, username, password)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("pgxpool.New: %w", err)
	}
	p.pool = pool
	return nil
}

// fetchCreds calls the Dapr secret store to get a fresh DB credential lease.
// Each call creates a new lease (new username + password) in the backend.
func (p *DaprPgPool) fetchCreds(ctx context.Context) (username, password string, err error) {
	secret, err := p.daprClient.GetSecret(ctx, p.secretStore, p.secretKey, nil)
	if err != nil {
		return "", "", fmt.Errorf("dapr GetSecret(%q, %q): %w", p.secretStore, p.secretKey, err)
	}
	username = secret["username"]
	password = secret["password"]
	if username == "" || password == "" {
		return "", "", fmt.Errorf("vault secret %q missing username or password", p.secretKey)
	}
	return username, password, nil
}
