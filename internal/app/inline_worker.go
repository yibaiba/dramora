package app

import (
	"context"
	"log/slog"

	"github.com/yibaiba/dramora/internal/jobs"
)

func StartInlineWorker(ctx context.Context, cfg Config, logger *slog.Logger, executor jobs.Executor) context.CancelFunc {
	workerCtx, cancel := context.WithCancel(ctx)
	if !cfg.InlineWorker {
		return cancel
	}
	if logger == nil {
		logger = slog.Default()
	}
	worker := jobs.NewWorker(logger.With("component", "inline_worker"), executor)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := worker.Run(workerCtx, cfg.WorkerQueues); err != nil {
			logger.Error("inline worker stopped", "error", err)
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
