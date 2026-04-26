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
	summary, err := w.executor.ProcessQueuedGenerationJobs(ctx, DefaultExecutionLimit)
	if err != nil {
		return summary, err
	}
	w.logger.Info("worker no-op batch complete",
		"processed", summary.Processed,
		"succeeded", summary.Succeeded,
		"failed", summary.Failed,
	)
	return summary, nil
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
