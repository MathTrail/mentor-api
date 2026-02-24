package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
)

// wsRe collapses consecutive whitespace (spaces, tabs, newlines) into a single
// space. gRPC metadata headers reject newline characters, so SQL must be
// flattened before being passed through the Dapr binding.
var wsRe = regexp.MustCompile(`\s+`)

func flattenSQL(sql string) string {
	return strings.TrimSpace(wsRe.ReplaceAllString(sql, " "))
}

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
//
// The Dapr PostgreSQL binding v1 returns positional arrays [[val1, val2], ...].
// To recover column names we wrap every query with json_agg(row_to_json(t))::text,
// which makes PostgreSQL serialise the result set as a single JSON text value.
// The binding then returns [[jsonText]] and we decode that inner JSON into
// []map[string]any — the same shape the rest of the codebase expects.
func (d *DaprDB) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	p, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("daprdb: marshal params: %w", err)
	}

	// Flatten whitespace so that multi-line SQL literals don't break gRPC
	// metadata headers (which reject newlines).
	flat := flattenSQL(sql)

	// Detect DML statements that cannot be used as subqueries in PostgreSQL.
	// We compare against the trimmed, uppercased SQL to handle leading whitespace.
	trimmed := strings.ToUpper(strings.TrimSpace(flat))
	var wrapped string
	if strings.HasPrefix(trimmed, "INSERT") ||
		strings.HasPrefix(trimmed, "UPDATE") ||
		strings.HasPrefix(trimmed, "DELETE") {
		wrapped = fmt.Sprintf(
			"WITH t AS (%s) SELECT COALESCE(json_agg(row_to_json(t))::text,'[]') FROM t",
			flat,
		)
	} else {
		wrapped = fmt.Sprintf(
			"SELECT COALESCE(json_agg(row_to_json(t))::text,'[]') FROM (%s) t",
			flat,
		)
	}

	resp, err := d.client.InvokeBinding(ctx, &dapr.InvokeBindingRequest{
		Name:      d.bindingName,
		Operation: "query",
		Metadata:  map[string]string{"sql": wrapped, "params": string(p)},
	})
	if err != nil {
		return nil, fmt.Errorf("daprdb: invoke binding: %w", err)
	}

	// Outer structure from v1: [[jsonText]] — one row, one text column.
	var outer [][]any
	if err := json.Unmarshal(resp.Data, &outer); err != nil {
		return nil, fmt.Errorf("daprdb: unmarshal outer: %w", err)
	}
	if len(outer) == 0 || len(outer[0]) == 0 {
		return []map[string]any{}, nil
	}
	jsonText, ok := outer[0][0].(string)
	if !ok {
		return nil, fmt.Errorf("daprdb: expected string from json_agg, got %T", outer[0][0])
	}

	rows := []map[string]any{}
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
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
		Metadata:  map[string]string{"sql": flattenSQL(sql), "params": string(p)},
	})
	if err != nil {
		return fmt.Errorf("daprdb: invoke binding: %w", err)
	}
	return nil
}

// Ping verifies the binding is reachable by executing a trivial statement.
// Uses exec (not query) to avoid JSON parsing overhead on a connectivity check.
func (d *DaprDB) Ping(ctx context.Context) error {
	return d.Exec(ctx, "SELECT 1")
}
