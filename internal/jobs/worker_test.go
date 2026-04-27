package jobs

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestWorkerRunOnceExecutesQueuedJobs(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		exportSummary:     ExecutionSummary{Processed: 1, Succeeded: 1},
		generationSummary: ExecutionSummary{Processed: 2, Succeeded: 2},
	}
	worker := NewWorker(slog.Default(), executor)

	summary, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !executor.called {
		t.Fatal("expected executor to be called")
	}
	if summary.Processed != 3 || summary.Succeeded != 3 {
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

func TestWorkerRunOnceReturnsGenerationSummaryWhenExportsFail(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		exportErr:         errors.New("export failed"),
		exportSummary:     ExecutionSummary{Processed: 1, Failed: 1},
		generationSummary: ExecutionSummary{Processed: 2, Succeeded: 2},
	}
	worker := NewWorker(slog.Default(), executor)

	summary, err := worker.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected export error")
	}
	if summary.Processed != 2 || summary.Succeeded != 2 || summary.Failed != 0 {
		t.Fatalf("expected generation-only summary, got %+v", summary)
	}
}

type fakeExecutor struct {
	called            bool
	exportErr         error
	exportSummary     ExecutionSummary
	generationSummary ExecutionSummary
}

func (e *fakeExecutor) ProcessQueuedGenerationJobs(_ context.Context, limit int) (ExecutionSummary, error) {
	e.called = true
	if limit != DefaultExecutionLimit {
		return ExecutionSummary{}, errors.New("unexpected limit")
	}
	return e.generationSummary, nil
}

func (e *fakeExecutor) ProcessQueuedExports(_ context.Context, limit int) (ExecutionSummary, error) {
	if limit != DefaultExecutionLimit {
		return ExecutionSummary{}, errors.New("unexpected limit")
	}
	return e.exportSummary, e.exportErr
}
