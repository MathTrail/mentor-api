package feedback

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MathTrail/mentor-api/internal/infra/postgres"
	"github.com/google/uuid"
)

const (
	unexpectedErrorFmt = "unexpected error: %v"
	expectedErrNilFmt  = "expected error, got nil"
)

// mockDB implements postgres.DB for repository tests.
type mockDB struct {
	queryFn func(ctx context.Context, sql string, params ...any) ([]map[string]any, error)
	execFn  func(ctx context.Context, sql string, params ...any) error
	pingFn  func(ctx context.Context) error
}

func (m *mockDB) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, params...)
	}
	return nil, nil
}

func (m *mockDB) Exec(ctx context.Context, sql string, params ...any) error {
	if m.execFn != nil {
		return m.execFn(ctx, sql, params...)
	}
	return nil
}

func (m *mockDB) Ping(ctx context.Context) error {
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return nil
}

// Compile-time interface check.
var _ postgres.DB = (*mockDB)(nil)

// --- Save tests ---

func TestSaveSuccess(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return []map[string]any{
				{
					"id":         id.String(),
					"created_at": now.Format(time.RFC3339),
				},
			}, nil
		},
	}

	repo := NewRepository(db)
	f := &Feedback{
		StudentID:           uuid.New(),
		Message:             "test message",
		PerceivedDifficulty: "ok",
		StrategySnapshot:    []byte(`{}`),
	}

	if err := repo.Save(context.Background(), f); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if f.ID != id {
		t.Errorf("ID: got %v, want %v", f.ID, id)
	}
	if f.CreatedAt.IsZero() {
		t.Error("CreatedAt should be non-zero after Save")
	}
}

func TestSaveDBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return nil, dbErr
		},
	}

	repo := NewRepository(db)
	f := &Feedback{StudentID: uuid.New(), Message: "test"}

	err := repo.Save(context.Background(), f)
	if err == nil {
		t.Fatal(expectedErrNilFmt)
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, dbErr)
	}
}

func TestSaveEmptyRows(t *testing.T) {
	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return []map[string]any{}, nil
		},
	}

	repo := NewRepository(db)
	f := &Feedback{StudentID: uuid.New(), Message: "test"}

	err := repo.Save(context.Background(), f)
	if err == nil {
		t.Fatal("expected error on empty rows, got nil")
	}
	if !strings.Contains(err.Error(), "no rows") {
		t.Errorf("error message: got %q, want to contain 'no rows'", err.Error())
	}
}

// --- GetLatestByStudent tests ---

func TestGetLatestByStudentSuccess(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	studentID := uuid.New()
	now := time.Now().UTC()

	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return []map[string]any{
				{
					"id":                   id1.String(),
					"student_id":           studentID.String(),
					"message":              "first",
					"perceived_difficulty": "ok",
					"strategy_snapshot":    `{}`,
					"created_at":           now.Format(time.RFC3339),
				},
				{
					"id":                   id2.String(),
					"student_id":           studentID.String(),
					"message":              "second",
					"perceived_difficulty": "hard",
					"strategy_snapshot":    `{}`,
					"created_at":           now.Add(-time.Minute).Format(time.RFC3339),
				},
			}, nil
		},
	}

	repo := NewRepository(db)
	results, err := repo.GetLatestByStudent(context.Background(), studentID, 10)
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(results) != 2 {
		t.Errorf("len: got %d, want 2", len(results))
	}
	if results[0].ID != id1 {
		t.Errorf("results[0].ID: got %v, want %v", results[0].ID, id1)
	}
	if results[1].ID != id2 {
		t.Errorf("results[1].ID: got %v, want %v", results[1].ID, id2)
	}
}

func TestGetLatestByStudentEmpty(t *testing.T) {
	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return []map[string]any{}, nil
		},
	}

	repo := NewRepository(db)
	results, err := repo.GetLatestByStudent(context.Background(), uuid.New(), 10)
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty slice, got len %d", len(results))
	}
}

func TestGetLatestByStudentDBError(t *testing.T) {
	dbErr := errors.New("timeout")
	db := &mockDB{
		queryFn: func(_ context.Context, _ string, _ ...any) ([]map[string]any, error) {
			return nil, dbErr
		},
	}

	repo := NewRepository(db)
	_, err := repo.GetLatestByStudent(context.Background(), uuid.New(), 5)
	if err == nil {
		t.Fatal(expectedErrNilFmt)
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, dbErr)
	}
}
