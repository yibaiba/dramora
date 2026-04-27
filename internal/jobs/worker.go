package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

var ErrExecutorRequired = errors.New("jobs executor is required")

type Executor interface {
	ProcessQueuedGenerationJobs(ctx context.Context, limit int) (ExecutionSummary, error)
	ProcessQueuedExports(ctx context.Context, limit int) (ExecutionSummary, error)
}

type Worker struct {
	logger   *slog.Logger
	executor Executor
}

func NewWorker(logger *slog.Logger, executor Executor) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{logger: logger, executor: executor}
}

func (w *Worker) RunOnce(ctx context.Context) (ExecutionSummary, error) {
	if w.executor == nil {
		return ExecutionSummary{}, ErrExecutorRequired
	}
	generationSummary, err := w.executor.ProcessQueuedGenerationJobs(ctx, DefaultExecutionLimit)
	if err != nil {
		return generationSummary, err
	}
	exportSummary, err := w.executor.ProcessQueuedExports(ctx, DefaultExecutionLimit)
	if err != nil {
		return generationSummary, err
	}
	summary := MergeExecutionSummaries(generationSummary, exportSummary)
	w.logger.Info("worker batch complete",
		"processed", summary.Processed,
		"succeeded", summary.Succeeded,
		"failed", summary.Failed,
	)
	return summary, nil
}

func MergeExecutionSummaries(items ...ExecutionSummary) ExecutionSummary {
	summary := ExecutionSummary{}
	for _, item := range items {
		summary.Processed += item.Processed
		summary.Succeeded += item.Succeeded
		summary.Failed += item.Failed
	}
	return summary
}

func (w *Worker) Run(ctx context.Context, queues []string) error {
	w.logger.Info("worker ready", "queues", queues)
	if _, err := w.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(DefaultPollInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}
