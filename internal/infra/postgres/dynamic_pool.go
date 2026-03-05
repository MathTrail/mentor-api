package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DynamicPool is a pgxpool.Pool wrapper that watches a mounted K8s Secret
// directory for credential changes and rotates the pool in-process without
// requiring a pod restart.
//
// K8s Secret volumes deliver updates via atomic symlink swaps: a new
// timestamped directory is created and the ..data symlink is atomically
// redirected to it. DynamicPool watches the parent directory for any event
// on the "..data" entry with a 200ms debounce, then re-reads username and
// password files and swaps the pool atomically.
type DynamicPool struct {
	ptr      atomic.Pointer[pgxpool.Pool]
	baseDSN  string // DSN without user/password (host, port, dbname, sslmode)
	credsDir string // directory containing "username" and "password" files
	logger   *zap.Logger
}

// NewDynamicPool creates a DynamicPool, reads initial credentials from credsDir,
// opens a connection pool, and starts a background watcher goroutine.
// Returns an error if credential files are absent or the initial pool cannot
// connect — the pod will restart (CrashLoopBackOff) until VSO creates the Secret.
func NewDynamicPool(ctx context.Context, baseDSN string, credsDir string, logger *zap.Logger) (*DynamicPool, error) {
	p := &DynamicPool{
		baseDSN:  baseDSN,
		credsDir: credsDir,
		logger:   logger,
	}

	pool, err := p.buildPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("initial pool: %w", err)
	}
	p.ptr.Store(pool)

	go p.watchCredentials(ctx)
	return p, nil
}

func (p *DynamicPool) readCredentials() (username, password string, err error) {
	raw, err := os.ReadFile(filepath.Join(p.credsDir, "username"))
	if err != nil {
		return "", "", fmt.Errorf("read username: %w", err)
	}
	username = strings.TrimSpace(string(raw))

	raw, err = os.ReadFile(filepath.Join(p.credsDir, "password"))
	if err != nil {
		return "", "", fmt.Errorf("read password: %w", err)
	}
	password = strings.TrimSpace(string(raw))

	return username, password, nil
}

func (p *DynamicPool) buildPool(ctx context.Context) (*pgxpool.Pool, error) {
	username, password, err := p.readCredentials()
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s user=%s password=%s", p.baseDSN, username, password)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return pool, nil
}

func (p *DynamicPool) reload(ctx context.Context) {
	newPool, err := p.buildPool(ctx)
	if err != nil {
		p.logger.Error("credential rotation failed, keeping current pool", zap.Error(err))
		return
	}

	oldPool := p.ptr.Swap(newPool)
	p.logger.Info("credentials rotated, pool swapped")

	// Close the old pool in the background. pgxpool.Close blocks until all
	// borrowed connections are returned, so in-flight queries finish gracefully.
	if oldPool != nil {
		go oldPool.Close()
	}
}

func (p *DynamicPool) watchCredentials(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		p.logger.Error("failed to create fsnotify watcher", zap.Error(err))
		return
	}
	defer watcher.Close()

	if err := watcher.Add(p.credsDir); err != nil {
		p.logger.Error("failed to watch credentials directory",
			zap.String("dir", p.credsDir), zap.Error(err))
		return
	}

	p.logger.Info("watching credentials directory for rotation",
		zap.String("dir", p.credsDir))

	var debounceTimer *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// K8s atomically swaps the ..data symlink when a Secret is updated.
			// React to any event on ..data regardless of Op (Create/Write/Rename
			// all observed across different kernel versions).
			if filepath.Base(event.Name) == "..data" {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
					p.reload(ctx)
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			p.logger.Error("fsnotify watcher error", zap.Error(err))
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// Query executes a SQL statement and returns result rows as a slice of maps.
// Column names are taken from the pgx FieldDescriptions.
func (p *DynamicPool) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	rows, err := p.ptr.Load().Query(ctx, sql, params...)
	if err != nil {
		return nil, fmt.Errorf("pgxpool query: %w", err)
	}
	return scanRows(rows)
}

// Exec executes a SQL statement that produces no result rows.
func (p *DynamicPool) Exec(ctx context.Context, sql string, params ...any) error {
	_, err := p.ptr.Load().Exec(ctx, sql, params...)
	if err != nil {
		return fmt.Errorf("pgxpool exec: %w", err)
	}
	return nil
}

// Ping verifies the connection pool is reachable.
func (p *DynamicPool) Ping(ctx context.Context) error {
	return p.ptr.Load().Ping(ctx)
}

// Close shuts down the active connection pool. Must be called once after the
// watchCredentials goroutine has exited (i.e. after ctx is cancelled).
func (p *DynamicPool) Close() {
	if pool := p.ptr.Load(); pool != nil {
		pool.Close()
	}
}
