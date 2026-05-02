package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Worker 不再注入 system bypass 上下文；每个 job 在 production service
	// 内部按真实归属派生 RoleWorker 上下文。
	//
	// 启动后台 goroutine 定期处理待结算记录（Pending Billing）
	go runPendingBillingWorker(ctx, logger, container.PendingBillingWorker)

	if err := worker.Run(ctx, cfg.WorkerQueues); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

// runPendingBillingWorker 定期处理待结算的扣费记录（积分计费系统重试机制）
func runPendingBillingWorker(ctx context.Context, logger *slog.Logger, pbWorker *service.PendingBillingWorker) {
	if pbWorker == nil {
		logger.Warn("pending billing worker is nil, skipping")
		return
	}

	// 每 5 分钟检查一次待结算记录
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("pending billing worker shutting down")
			return
		case <-ticker.C:
			processed, succeeded, failed, err := pbWorker.ProcessOnce(ctx, 20)
			if err != nil {
				logger.Error("pending billing worker error", "error", err)
			} else if processed > 0 {
				logger.Info("pending billing worker processed batch",
					"processed", processed,
					"succeeded", succeeded,
					"failed", failed,
				)
			}
		}
	}
}
