package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/yibaiba/dramora/internal/jobs"
)

func TestStartInlineWorkerProcessesQueuedJobsWhenEnabled(t *testing.T) {
	executor := &inlineWorkerTestExecutor{called: make(chan struct{})}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cancel := StartInlineWorker(context.Background(), Config{
		InlineWorker: true,
		WorkerQueues: []string{string(jobs.QueueDefault)},
	}, logger, executor)
	defer cancel()

	select {
	case <-executor.called:
	case <-time.After(time.Second):
		t.Fatal("expected inline worker to process immediately")
	}
}

func TestStartInlineWorkerSkipsWhenDisabled(t *testing.T) {
	executor := &inlineWorkerTestExecutor{called: make(chan struct{})}
	cancel := StartInlineWorker(context.Background(), Config{InlineWorker: false}, nil, executor)
	defer cancel()

	select {
	case <-executor.called:
		t.Fatal("expected disabled inline worker not to process")
	case <-time.After(20 * time.Millisecond):
	}
}

type inlineWorkerTestExecutor struct {
	once   sync.Once
	called chan struct{}
}

func (e *inlineWorkerTestExecutor) ProcessQueuedGenerationJobs(
	_ context.Context,
	_ int,
) (jobs.ExecutionSummary, error) {
	e.once.Do(func() { close(e.called) })
	return jobs.ExecutionSummary{Processed: 1, Succeeded: 1}, nil
}

func (e *inlineWorkerTestExecutor) ProcessQueuedExports(_ context.Context, _ int) (jobs.ExecutionSummary, error) {
	return jobs.ExecutionSummary{}, nil
}
