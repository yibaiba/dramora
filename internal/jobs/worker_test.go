package jobs

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestWorkerRunOnceExecutesQueuedJobs(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{summary: ExecutionSummary{Processed: 1, Succeeded: 1}}
	worker := NewWorker(slog.Default(), executor)

	summary, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !executor.called {
		t.Fatal("expected executor to be called")
	}
	if summary.Processed != 1 || summary.Succeeded != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestWorkerRunOnceRequiresExecutor(t *testing.T) {
	t.Parallel()

	_, err := NewWorker(slog.Default(), nil).RunOnce(context.Background())
	if !errors.Is(err, ErrExecutorRequired) {
		t.Fatalf("expected executor required error, got %v", err)
	}
}

type fakeExecutor struct {
	called  bool
	summary ExecutionSummary
}

func (e *fakeExecutor) ProcessQueuedGenerationJobs(_ context.Context, limit int) (ExecutionSummary, error) {
	e.called = true
	if limit != DefaultExecutionLimit {
		return ExecutionSummary{}, errors.New("unexpected limit")
	}
	return e.summary, nil
}
