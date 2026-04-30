package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yibaiba/dramora/internal/app"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	container, err := app.NewContainer(ctx, cfg, logger)
	if err != nil {
		logger.Error("create container", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	worker := jobs.NewWorker(logger, container.ProductionService)

	systemCtx := service.WithSystemAuthContext(ctx)
	if err := worker.Run(systemCtx, cfg.WorkerQueues); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
