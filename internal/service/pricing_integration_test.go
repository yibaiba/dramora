package service

import (
	"context"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// TestWalletDebitOperationIdempotency 测试 DebitOperation 的幂等性。
// 相同的 refID 不应该重复扣费。
func TestWalletDebitOperationIdempotency(t *testing.T) {
	// 构建测试环境
	walletRepo := repo.NewMemoryWalletRepository()
	pendingBillingRepo := repo.NewMemoryPendingBillingRepository()

	// 创建钱包并充值
	orgID := "test-org-1"
	userID := "test-user-1"
	ctx := WithRequestAuthContext(context.Background(), RequestAuthContext{
		UserID:         userID,
		OrganizationID: orgID,
	})

	// 充值 1000 积分
	_, _, err := walletRepo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: orgID,
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         1000,
		Reason:         "initial top-up",
		RefType:        "manual",
		RefID:          "topup-1",
		ActorUserID:    userID,
		TransactionID:  "tx-1",
	})
	if err != nil {
		t.Fatalf("failed to top-up wallet: %v", err)
	}

	// 创建 WalletService
	walletSvc := NewWalletService(walletRepo, nil)
	walletSvc.SetPendingBillingRepository(pendingBillingRepo)

	// 第一次扣费
	genJobID := "gen-job-1"
	tx1, err := walletSvc.DebitOperation(ctx, domain.OperationTypeStoryAnalysis, "generation_job_id", genJobID)
	if err != nil {
		t.Fatalf("first debit failed: %v", err)
	}
	if tx1.Amount != 50 {
		t.Errorf("expected amount 50, got %d", tx1.Amount)
	}

	// 验证余额减少
	snap, err := walletSvc.GetWallet(ctx)
	if err != nil {
		t.Fatalf("failed to get wallet: %v", err)
	}
	if snap.Wallet.Balance != 950 {
		t.Errorf("expected balance 950, got %d", snap.Wallet.Balance)
	}

	// 第二次扣费相同的操作 - 应该返回 ErrAlreadyDebited
	_, err = walletSvc.DebitOperation(ctx, domain.OperationTypeStoryAnalysis, "generation_job_id", genJobID)
	if err == nil || !isErrAlreadyDebited(err) {
		t.Fatalf("expected ErrAlreadyDebited, got: %v", err)
	}

	// 验证余额没有再减少
	snap, err = walletSvc.GetWallet(ctx)
	if err != nil {
		t.Fatalf("failed to get wallet after second debit: %v", err)
	}
	if snap.Wallet.Balance != 950 {
		t.Errorf("balance should remain 950, got %d", snap.Wallet.Balance)
	}

	// 验证只有一条交易记录
	page, err := walletSvc.ListTransactions(ctx, 100, 0, nil)
	if err != nil {
		t.Fatalf("failed to list transactions: %v", err)
	}
	if len(page.Transactions) != 2 { // topup + debit
		t.Errorf("expected 2 transactions (topup + debit), got %d", len(page.Transactions))
	}
}

// TestPendingBillingWorkerRetryOnInsufficientBalance 测试待结算重试机制。
// 余额不足时应该创建待结算记录，worker 应该在余额充足后重试成功。
func TestPendingBillingWorkerRetryOnInsufficientBalance(t *testing.T) {
	walletRepo := repo.NewMemoryWalletRepository()
	pendingBillingRepo := repo.NewMemoryPendingBillingRepository()

	orgID := "test-org-2"
	userID := "test-user-2"
	ctx := WithRequestAuthContext(context.Background(), RequestAuthContext{
		UserID:         userID,
		OrganizationID: orgID,
	})

	// 仅充值 30 积分（不足以支付 story_analysis 的 50 积分）
	_, _, err := walletRepo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: orgID,
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         30,
		Reason:         "insufficient top-up",
		RefType:        "manual",
		RefID:          "topup-2",
		ActorUserID:    userID,
		TransactionID:  "tx-2",
	})
	if err != nil {
		t.Fatalf("failed to top-up wallet: %v", err)
	}

	walletSvc := NewWalletService(walletRepo, nil)
	walletSvc.SetPendingBillingRepository(pendingBillingRepo)

	// 尝试扣费 - 应该失败但创建待结算记录
	genJobID := "gen-job-2"
	_, err = walletSvc.DebitOperation(ctx, domain.OperationTypeStoryAnalysis, "generation_job_id", genJobID)
	if err == nil || !isErrInsufficientBalance(err) {
		t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
	}

	// 验证待结算记录已创建
	pending, err := pendingBillingRepo.GetByRef(ctx, orgID, "story_analysis", genJobID)
	if err != nil {
		t.Fatalf("failed to fetch pending billing: %v", err)
	}
	if pending == nil || pending.Status != domain.PendingBillingStatusPending {
		t.Errorf("expected pending billing with status=pending, got: %v", pending)
	}

	// 验证钱包余额仍为 30
	snap, err := walletSvc.GetWallet(ctx)
	if err != nil {
		t.Fatalf("failed to get wallet: %v", err)
	}
	if snap.Wallet.Balance != 30 {
		t.Errorf("expected balance 30, got %d", snap.Wallet.Balance)
	}

	// 充值更多积分使得总余额足够
	_, _, err = walletRepo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: orgID,
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         50,
		Reason:         "additional top-up",
		RefType:        "manual",
		RefID:          "topup-3",
		ActorUserID:    userID,
		TransactionID:  "tx-3",
	})
	if err != nil {
		t.Fatalf("failed to add more credits: %v", err)
	}

	// 运行 worker 重试
	worker := NewPendingBillingWorker(nil, pendingBillingRepo, walletSvc)
	processed, succeeded, failed, err := worker.ProcessOnce(ctx, 10)
	if err != nil {
		t.Fatalf("worker ProcessOnce failed: %v", err)
	}

	if processed != 1 {
		t.Errorf("expected 1 processed, got %d", processed)
	}
	if succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", succeeded)
	}
	if failed != 0 {
		t.Errorf("expected 0 failed, got %d", failed)
	}

	// 验证待结算记录已标记为 resolved
	pending, err = pendingBillingRepo.GetByRef(ctx, orgID, "story_analysis", genJobID)
	if err != nil {
		t.Fatalf("failed to fetch pending billing after retry: %v", err)
	}
	if pending == nil || pending.Status != domain.PendingBillingStatusResolved {
		t.Errorf("expected pending billing with status=resolved, got: %v", pending)
	}

	// 验证钱包余额现在为 30（原始）+ 50（新增）- 50（扣费）= 30
	snap, err = walletSvc.GetWallet(ctx)
	if err != nil {
		t.Fatalf("failed to get wallet after retry: %v", err)
	}
	if snap.Wallet.Balance != 30 {
		t.Errorf("expected final balance 30, got %d", snap.Wallet.Balance)
	}
}

// TestDebitOperationWithDifferentOperationTypes 测试不同操作类型的扣费。
func TestDebitOperationWithDifferentOperationTypes(t *testing.T) {
	walletRepo := repo.NewMemoryWalletRepository()
	pendingBillingRepo := repo.NewMemoryPendingBillingRepository()

	orgID := "test-org-3"
	userID := "test-user-3"
	ctx := WithRequestAuthContext(context.Background(), RequestAuthContext{
		UserID:         userID,
		OrganizationID: orgID,
	})

	// 充值足够的积分
	_, _, err := walletRepo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: orgID,
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         5000,
		Reason:         "test top-up",
		RefType:        "manual",
		RefID:          "topup-4",
		ActorUserID:    userID,
		TransactionID:  "tx-4",
	})
	if err != nil {
		t.Fatalf("failed to top-up wallet: %v", err)
	}

	walletSvc := NewWalletService(walletRepo, nil)
	walletSvc.SetPendingBillingRepository(pendingBillingRepo)

	// 测试各种操作类型的扣费
	// 注：OperationTypeChat 不再用 DebitOperation，使用 DebitChatOperation（基于 token）
	tests := []struct {
		opType   domain.OperationType
		expected int64
	}{
		// {domain.OperationTypeChat, 1}, // Chat 使用 token 计费，见下方
		{domain.OperationTypeImageGeneration, 100},
		{domain.OperationTypeVideoGeneration, 200},
		{domain.OperationTypeStoryboardEdit, 5},
	}

	initialBalance := int64(5000)
	for i, tc := range tests {
		refID := "job-" + string(rune(i))
		_, err := walletSvc.DebitOperation(ctx, tc.opType, "generation_job_id", refID)
		if err != nil {
			t.Fatalf("failed to debit %s: %v", tc.opType, err)
		}

		initialBalance -= tc.expected
	}

	// 测试 chat 操作（基于 token）
	// 1000 tokens = 10 积分；(500 + 600 + 999) / 1000 * 10 = 2 * 10 = 20
	_, err = walletSvc.DebitChatOperation(ctx, 500, 600, "chat-msg-1")
	if err != nil {
		t.Fatalf("failed to debit chat: %v", err)
	}
	initialBalance -= 20

	// 验证最终余额
	snap, err := walletSvc.GetWallet(ctx)
	if err != nil {
		t.Fatalf("failed to get wallet: %v", err)
	}

	if snap.Wallet.Balance != initialBalance {
		t.Errorf("expected balance %d, got %d", initialBalance, snap.Wallet.Balance)
	}
}

// 辅助函数
func isErrAlreadyDebited(err error) bool {
	return err != nil && err == ErrAlreadyDebited
}

func isErrInsufficientBalance(err error) bool {
	return err != nil && err == ErrInsufficientBalance
}
