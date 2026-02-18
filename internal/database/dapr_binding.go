package database

import (
	"context"
	"encoding/json"
	"fmt"

	dapr "github.com/dapr/go-sdk/client"
)

// DaprDB executes SQL via a Dapr PostgreSQL output binding.
// The binding reads its connection string from a K8s Secret managed by ESO,
// so credential rotation is transparent to the application.
type DaprDB struct {
	client      dapr.Client
	bindingName string
}

// NewDaprDB creates a DaprDB backed by the named Dapr binding component.
func NewDaprDB(client dapr.Client, bindingName string) *DaprDB {
	return &DaprDB{client: client, bindingName: bindingName}
}

// Query executes a SQL statement and returns the result rows as a slice of maps.
// Use this for SELECT statements and for INSERT/UPDATE … RETURNING.
func (d *DaprDB) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	p, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("daprdb: marshal params: %w", err)
	}

	resp, err := d.client.InvokeBinding(ctx, &dapr.InvokeBindingRequest{
		Name:      d.bindingName,
		Operation: "query",
		Data:      []byte(sql),
		Metadata:  map[string]string{"params": string(p)},
	})
	if err != nil {
		return nil, fmt.Errorf("daprdb: invoke binding: %w", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Data, &rows); err != nil {
		return nil, fmt.Errorf("daprdb: unmarshal rows: %w", err)
	}
	return rows, nil
}

// Exec executes a SQL statement that produces no result rows.
// Use this for INSERT/UPDATE/DELETE without RETURNING.
func (d *DaprDB) Exec(ctx context.Context, sql string, params ...any) error {
	p, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("daprdb: marshal params: %w", err)
	}

	_, err = d.client.InvokeBinding(ctx, &dapr.InvokeBindingRequest{
		Name:      d.bindingName,
		Operation: "exec",
		Data:      []byte(sql),
		Metadata:  map[string]string{"params": string(p)},
	})
	if err != nil {
		return fmt.Errorf("daprdb: invoke binding: %w", err)
	}
	return nil
}

// Ping verifies the binding is reachable by executing a trivial query.
func (d *DaprDB) Ping(ctx context.Context) error {
	_, err := d.Query(ctx, "SELECT 1")
	return err
}
