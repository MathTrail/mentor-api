package postgres

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockRows is a minimal pgx.Rows implementation for unit-testing scanRows.
type mockRows struct {
	fields  []pgconn.FieldDescription
	data    [][]any
	idx     int
	valErr  error
	rowsErr error
}

func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return m.rowsErr }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return m.fields }
func (m *mockRows) Scan(_ ...any) error                          { return nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Conn() *pgx.Conn                              { return nil }

func (m *mockRows) Next() bool {
	if m.idx < len(m.data) {
		m.idx++
		return true
	}
	return false
}

func (m *mockRows) Values() ([]any, error) {
	if m.valErr != nil {
		return nil, m.valErr
	}
	return m.data[m.idx-1], nil
}

func TestScanRows_Empty(t *testing.T) {
	rows := &mockRows{}
	got, err := scanRows(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestScanRows_StringColumns(t *testing.T) {
	rows := &mockRows{
		fields: []pgconn.FieldDescription{
			{Name: "name"},
			{Name: "role"},
		},
		data: [][]any{
			{"Alice", "admin"},
			{"Bob", "user"},
		},
	}
	got, err := scanRows(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0]["name"] != "Alice" {
		t.Errorf("row[0].name = %v, want Alice", got[0]["name"])
	}
	if got[1]["role"] != "user" {
		t.Errorf("row[1].role = %v, want user", got[1]["role"])
	}
}

func TestScanRows_UUIDColumn(t *testing.T) {
	id := uuid.New()
	var raw [16]byte
	copy(raw[:], id[:])

	rows := &mockRows{
		fields: []pgconn.FieldDescription{
			{Name: "id", DataTypeOID: pgtype.UUIDOID},
		},
		data: [][]any{
			{raw},
		},
	}
	got, err := scanRows(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := got[0]["id"].(uuid.UUID)
	if !ok {
		t.Fatalf("expected uuid.UUID, got %T", got[0]["id"])
	}
	if v != id {
		t.Errorf("id = %v, want %v", v, id)
	}
}

func TestScanRows_ValuesError(t *testing.T) {
	scanErr := errors.New("scan failed")
	rows := &mockRows{
		fields: []pgconn.FieldDescription{{Name: "x"}},
		data:   [][]any{{"value"}},
		valErr: scanErr,
	}
	_, err := scanRows(rows)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, scanErr) {
		t.Errorf("error = %v, want to wrap %v", err, scanErr)
	}
}

func TestScanRows_RowsErr(t *testing.T) {
	iterErr := errors.New("iteration failed")
	rows := &mockRows{
		rowsErr: iterErr,
	}
	_, err := scanRows(rows)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, iterErr) {
		t.Errorf("error = %v, want to wrap %v", err, iterErr)
	}
}
