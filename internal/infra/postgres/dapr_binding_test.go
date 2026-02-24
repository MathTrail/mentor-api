package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	dapr "github.com/dapr/go-sdk/client"
)

// mockDaprClient embeds the dapr.Client interface (with a nil value) so that
// only InvokeBinding needs to be implemented. Calls to any other method will
// panic with a nil-pointer dereference, which is acceptable in tests that
// only exercise the binding path.
type mockDaprClient struct {
	dapr.Client
	invokeFn func(ctx context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error)
}

func (m *mockDaprClient) InvokeBinding(ctx context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
	return m.invokeFn(ctx, in)
}

// bindingResponse builds the [[jsonText]] wire format that DaprDB.Query expects.
func bindingResponse(rows []map[string]any) *dapr.BindingEvent {
	inner, _ := json.Marshal(rows)
	outer, _ := json.Marshal([][]any{{string(inner)}})
	return &dapr.BindingEvent{Data: outer}
}

// --- flattenSQL (white-box: package postgres) ---

func TestFlattenSQL_CollapsesWhitespace(t *testing.T) {
	got := flattenSQL("SELECT\n  *\n  FROM\t t")
	want := "SELECT * FROM t"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFlattenSQL_TrimsEdges(t *testing.T) {
	got := flattenSQL("  SELECT 1  ")
	want := "SELECT 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Query ---

func TestQuery_SELECT_WrapsAsSubquery(t *testing.T) {
	var capturedSQL string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedSQL = in.Metadata["sql"]
			return bindingResponse(nil), nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, _ = db.Query(context.Background(), "SELECT id FROM feedback WHERE id = $1", "abc")

	if !strings.Contains(capturedSQL, "FROM (SELECT id FROM feedback") {
		t.Errorf("SELECT wrap: expected FROM (...) t pattern, got %q", capturedSQL)
	}
	if strings.Contains(capturedSQL, "WITH t AS") {
		t.Errorf("SELECT should not use CTE wrapping, got %q", capturedSQL)
	}
}

func TestQuery_INSERT_WrapsAsCTE(t *testing.T) {
	var capturedSQL string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedSQL = in.Metadata["sql"]
			return bindingResponse(nil), nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, _ = db.Query(context.Background(), "INSERT INTO feedback (message) VALUES ($1) RETURNING id", "hi")

	if !strings.HasPrefix(capturedSQL, "WITH t AS (INSERT") {
		t.Errorf("INSERT wrap: expected WITH t AS (INSERT ...) pattern, got %q", capturedSQL)
	}
}

func TestQuery_UPDATE_WrapsAsCTE(t *testing.T) {
	var capturedSQL string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedSQL = in.Metadata["sql"]
			return bindingResponse(nil), nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, _ = db.Query(context.Background(), "UPDATE feedback SET message = $1 WHERE id = $2 RETURNING id", "new", "abc")

	if !strings.HasPrefix(capturedSQL, "WITH t AS (UPDATE") {
		t.Errorf("UPDATE wrap: expected WITH t AS (UPDATE ...) pattern, got %q", capturedSQL)
	}
}

func TestQuery_UsesQueryOperation(t *testing.T) {
	var capturedOp string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedOp = in.Operation
			return bindingResponse(nil), nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, _ = db.Query(context.Background(), "SELECT 1")
	if capturedOp != "query" {
		t.Errorf("operation: got %q, want %q", capturedOp, "query")
	}
}

func TestQuery_DecodesRows(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, _ *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			return bindingResponse([]map[string]any{{"id": id, "message": "hello"}}), nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	rows, err := db.Query(context.Background(), "SELECT id, message FROM feedback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows): got %d, want 1", len(rows))
	}
	if rows[0]["id"] != id {
		t.Errorf("rows[0][id]: got %v, want %q", rows[0]["id"], id)
	}
}

func TestQuery_EmptyOuter_ReturnsEmpty(t *testing.T) {
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, _ *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			return &dapr.BindingEvent{Data: []byte("[]")}, nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	rows, err := db.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected empty result, got %d rows", len(rows))
	}
}

func TestQuery_InvokeBindingError(t *testing.T) {
	bindingErr := errors.New("dapr sidecar not reachable")
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, _ *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			return nil, bindingErr
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, err := db.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, bindingErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, bindingErr)
	}
}

func TestQuery_MalformedOuterJSON(t *testing.T) {
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, _ *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			return &dapr.BindingEvent{Data: []byte("not-json")}, nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	_, err := db.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal outer") {
		t.Errorf("error: got %q, want to contain 'unmarshal outer'", err.Error())
	}
}

// --- Exec ---

func TestExec_UsesExecOperation(t *testing.T) {
	var capturedOp string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedOp = in.Operation
			return &dapr.BindingEvent{}, nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	if err := db.Exec(context.Background(), "DELETE FROM feedback WHERE id = $1", "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOp != "exec" {
		t.Errorf("operation: got %q, want %q", capturedOp, "exec")
	}
}

func TestExec_Error(t *testing.T) {
	execErr := errors.New("exec failed")
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, _ *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			return nil, execErr
		},
	}
	db := NewDaprDB(client, "test-binding")
	err := db.Exec(context.Background(), "DELETE FROM feedback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, execErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, execErr)
	}
}

// --- Ping ---

func TestPing_CallsSelectOne(t *testing.T) {
	var capturedSQL string
	client := &mockDaprClient{
		invokeFn: func(_ context.Context, in *dapr.InvokeBindingRequest) (*dapr.BindingEvent, error) {
			capturedSQL = in.Metadata["sql"]
			return &dapr.BindingEvent{}, nil
		},
	}
	db := NewDaprDB(client, "test-binding")
	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSQL != "SELECT 1" {
		t.Errorf("Ping SQL: got %q, want %q", capturedSQL, "SELECT 1")
	}
}
