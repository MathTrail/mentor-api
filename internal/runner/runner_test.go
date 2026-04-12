package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MathTrail/mentor-api/internal/runner"
	runnermocks "github.com/MathTrail/mentor-api/internal/runner/mocks"
	"github.com/stretchr/testify/mock"
)

func TestRunGroup_AllSucceed(t *testing.T) {
	w1 := runnermocks.NewMockWorker(t)
	w2 := runnermocks.NewMockWorker(t)
	w1.EXPECT().Start(mock.Anything).Return(nil)
	w2.EXPECT().Start(mock.Anything).Return(nil)

	if err := runner.RunGroup(context.Background(), w1, w2); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestRunGroup_OneErrors(t *testing.T) {
	boom := errors.New("worker boom")

	w1 := runnermocks.NewMockWorker(t)
	w2 := runnermocks.NewMockWorker(t)
	w1.EXPECT().Start(mock.Anything).Return(boom)
	// w2 may or may not be called depending on goroutine scheduling.
	w2.EXPECT().Start(mock.Anything).Return(nil).Maybe()

	err := runner.RunGroup(context.Background(), w1, w2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error: got %v, want to contain %v", err, boom)
	}
}

func TestRunGroup_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before RunGroup

	w := runnermocks.NewMockWorker(t)
	// Worker receives already-cancelled errgroup ctx and returns nil (graceful shutdown).
	w.EXPECT().Start(mock.Anything).Return(nil)

	if err := runner.RunGroup(ctx, w); err != nil {
		t.Errorf("expected nil on clean shutdown, got: %v", err)
	}
}
