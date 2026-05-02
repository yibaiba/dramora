package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// PendingBillingWorker 负责重试失败的扣费操作。
type PendingBillingWorker struct {
	logger             *slog.Logger
	pendingBillingRepo repo.PendingBillingRepository
	walletSvc          *WalletService
}

// NewPendingBillingWorker 创建待结算重试 worker。
func NewPendingBillingWorker(
	logger *slog.Logger,
	pendingBillingRepo repo.PendingBillingRepository,
	walletSvc *WalletService,
) *PendingBillingWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &PendingBillingWorker{
		logger:             logger,
		pendingBillingRepo: pendingBillingRepo,
		walletSvc:          walletSvc,
	}
}

// ProcessOnce 一次性处理所有待重试的待结算记录。
// 返回处理数、成功数、失败数。
func (w *PendingBillingWorker) ProcessOnce(ctx context.Context, limit int) (processed, succeeded, failed int, err error) {
	if limit <= 0 {
		limit = 10
	}

	// 获取所有待重试的记录（status=pending 且 retry_count < max_retries）
	pendingBillings, err := w.pendingBillingRepo.GetPending(ctx, limit)
	if err != nil {
		w.logger.Error("failed to fetch pending billings", "error", err)
		return 0, 0, 0, err
	}

	if len(pendingBillings) == 0 {
		return 0, 0, 0, nil
	}

	for _, pb := range pendingBillings {
		processed++
		if err := w.processPendingBilling(ctx, pb); err != nil {
			w.logger.Error("failed to process pending billing",
				"id", pb.ID,
				"ref_id", pb.RefID,
				"retry_count", pb.RetryCount,
				"error", err,
			)
			failed++
		} else {
			succeeded++
		}
	}

	return processed, succeeded, failed, nil
}

// processPendingBilling 处理单条待结算记录。
func (w *PendingBillingWorker) processPendingBilling(ctx context.Context, pb *domain.PendingBilling) error {
	if pb == nil {
		return errors.New("pending billing is nil")
	}

	// 检查是否超过最大重试次数
	if pb.RetryCount >= pb.MaxRetries {
		return w.markFailed(ctx, pb, "max retries exceeded")
	}

	// 尝试扣费（使用 InternalDebit）
	params := CreditParams{
		Amount:  pb.Amount,
		Reason:  fmt.Sprintf("pending billing debit: %s", pb.OperationType),
		RefType: pb.RefType,
		RefID:   pb.RefID,
	}
	_, err := w.walletSvc.InternalDebit(ctx, pb.OrganizationID, params)
	if err == nil {
		// 成功，标记为 resolved
		return w.markResolved(ctx, pb)
	}

	// 扣费失败，检查错误类型
	if errors.Is(err, ErrAlreadyDebited) {
		// 已经扣费过，标记为 resolved
		return w.markResolved(ctx, pb)
	}

	if errors.Is(err, ErrInsufficientBalance) {
		// 余额仍然不足，增加重试计数，保持 pending
		return w.incrementRetry(ctx, pb, "insufficient balance")
	}

	// 其他错误，记录并增加重试
	return w.incrementRetry(ctx, pb, err.Error())
}

// markResolved 标记待结算记录为已解决。
func (w *PendingBillingWorker) markResolved(ctx context.Context, pb *domain.PendingBilling) error {
	pb.Status = domain.PendingBillingStatusResolved
	pb.UpdatedAt = time.Now().Unix()
	if err := w.pendingBillingRepo.Update(ctx, pb); err != nil {
		w.logger.Error("failed to mark pending billing as resolved",
			"id", pb.ID,
			"error", err,
		)
		return err
	}
	w.logger.Info("pending billing resolved",
		"id", pb.ID,
		"ref_id", pb.RefID,
		"amount", pb.Amount,
	)
	return nil
}

// markFailed 标记待结算记录为最终失败（需运营处理）。
func (w *PendingBillingWorker) markFailed(ctx context.Context, pb *domain.PendingBilling, reason string) error {
	pb.Status = domain.PendingBillingStatusFailed
	pb.LastErrorMsg = reason
	pb.UpdatedAt = time.Now().Unix()
	if err := w.pendingBillingRepo.Update(ctx, pb); err != nil {
		w.logger.Error("failed to mark pending billing as failed",
			"id", pb.ID,
			"error", err,
		)
		return err
	}
	w.logger.Warn("pending billing marked as failed",
		"id", pb.ID,
		"ref_id", pb.RefID,
		"reason", reason,
		"retry_count", pb.RetryCount,
	)
	return nil
}

// incrementRetry 增加重试计数。
func (w *PendingBillingWorker) incrementRetry(ctx context.Context, pb *domain.PendingBilling, errorMsg string) error {
	pb.RetryCount++
	pb.LastErrorMsg = errorMsg
	pb.UpdatedAt = time.Now().Unix()
	// 若超过最大重试次数，直接标记为失败
	if pb.RetryCount >= pb.MaxRetries {
		pb.Status = domain.PendingBillingStatusFailed
		w.logger.Warn("pending billing max retries reached",
			"id", pb.ID,
			"ref_id", pb.RefID,
			"retry_count", pb.RetryCount,
		)
	} else {
		pb.Status = domain.PendingBillingStatusPending
	}

	if err := w.pendingBillingRepo.Update(ctx, pb); err != nil {
		w.logger.Error("failed to update pending billing retry",
			"id", pb.ID,
			"error", err,
		)
		return err
	}
	return nil
}
