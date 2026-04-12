package onboarding

import (
	"context"
	"errors"
	"testing"
	"time"

	pgmocks "github.com/MathTrail/mentor-api/internal/infra/postgres/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestUpsertSuccess(t *testing.T) {
	db := pgmocks.NewMockDB(t)
	// Exec(ctx, query, studentID, eventID, occurredAt) — 5 args.
	db.EXPECT().Exec(
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	repo := NewRepository(db)
	err := repo.Upsert(context.Background(), uuid.New(), "evt-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpsertDBError(t *testing.T) {
	dbErr := errors.New("connection refused")

	db := pgmocks.NewMockDB(t)
	db.EXPECT().Exec(
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything,
	).Return(dbErr)

	repo := NewRepository(db)
	err := repo.Upsert(context.Background(), uuid.New(), "evt-2", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, dbErr)
	}
}
