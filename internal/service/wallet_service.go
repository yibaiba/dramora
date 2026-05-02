package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// ErrInsufficientBalance 暴露给 HTTP 层用于映射 422 / 自定义提示。
var ErrInsufficientBalance = repo.ErrInsufficientBalance
var ErrAlreadyDebited = errors.New("wallet: operation already debited")

// WalletService 围绕 WalletRepository 实现按组织上下文的钱包能力。
type WalletService struct {
	repo                repo.WalletRepository
	notificationSvc     *NotificationService
	pendingBillingRepo  repo.PendingBillingRepository
	operationCostRepo   repo.OperationCostRepository
}

// NewWalletService 构造 WalletService；repo 为 nil 时所有方法均返回 ErrUnauthorized。
func NewWalletService(r repo.WalletRepository, notifSvc *NotificationService) *WalletService {
	return &WalletService{
		repo:              r,
		notificationSvc:   notifSvc,
		operationCostRepo: repo.NewMemoryOperationCostRepository(), // 默认使用内存仓库
	}
}

// SetPendingBillingRepository 设置待结算仓库（用于扣费失败场景）。
func (s *WalletService) SetPendingBillingRepository(pbr repo.PendingBillingRepository) {
	if s != nil {
		s.pendingBillingRepo = pbr
	}
}

// SetOperationCostRepository 设置操作成本仓库。
func (s *WalletService) SetOperationCostRepository(ocr repo.OperationCostRepository) {
	if s != nil {
		s.operationCostRepo = ocr
	}
}

// GetOperationCostRepository 返回操作成本仓库（供 Admin handlers 使用）。
func (s *WalletService) GetOperationCostRepository() repo.OperationCostRepository {
	if s != nil {
		return s.operationCostRepo
	}
	return nil
}

// WalletSnapshot 是 GET /wallet 的统一读模型。
type WalletSnapshot struct {
	Wallet             domain.Wallet
	RecentTransactions []domain.WalletTransaction
}

func (s *WalletService) requireAuth(ctx context.Context) (RequestAuthContext, error) {
	if s == nil || s.repo == nil {
		return RequestAuthContext{}, ErrUnauthorized
	}
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return RequestAuthContext{}, ErrUnauthorized
	}
	return auth, nil
}

// getOperationCostFromDB 获取操作成本，优先查询数据库，回退到常量。
// 支持组织隔离定价。
func (s *WalletService) getOperationCostFromDB(ctx context.Context, orgID string, opType domain.OperationType) (int64, error) {
	if s.operationCostRepo == nil {
		// 回退到常量
		return domain.GetOperationCost(opType)
	}

	row, err := s.operationCostRepo.GetCost(ctx, orgID, opType)
	if err != nil {
		// 查询出错，回退到常量
		return domain.GetOperationCost(opType)
	}

	if row == nil {
		// 未找到，回退到常量
		return domain.GetOperationCost(opType)
	}

	return row.CreditsCost, nil
}

// GetWallet 返回当前组织的余额与最近 10 条流水。
func (s *WalletService) GetWallet(ctx context.Context) (WalletSnapshot, error) {
	auth, err := s.requireAuth(ctx)
	if err != nil {
		return WalletSnapshot{}, err
	}
	w, err := s.repo.GetWallet(ctx, auth.OrganizationID)
	if err != nil {
		return WalletSnapshot{}, err
	}
	w.OrganizationID = auth.OrganizationID
	page, err := s.repo.ListTransactions(ctx, repo.WalletTransactionFilter{
		OrganizationID: auth.OrganizationID,
		Limit:          10,
	})
	if err != nil {
		return WalletSnapshot{}, err
	}
	return WalletSnapshot{Wallet: w, RecentTransactions: page.Transactions}, nil
}

// ListTransactions 返回当前组织的流水分页结果。
func (s *WalletService) ListTransactions(ctx context.Context, limit, offset int, kinds []string) (repo.WalletTransactionPage, error) {
	auth, err := s.requireAuth(ctx)
	if err != nil {
		return repo.WalletTransactionPage{}, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	for i, k := range kinds {
		k = strings.TrimSpace(k)
		if !domain.IsValidWalletKind(k) {
			return repo.WalletTransactionPage{}, fmt.Errorf("wallet: invalid kind %q", k)
		}
		kinds[i] = k
	}
	return s.repo.ListTransactions(ctx, repo.WalletTransactionFilter{
		OrganizationID: auth.OrganizationID,
		Kinds:          kinds,
		Limit:          limit,
		Offset:         offset,
	})
}

// CreditParams 描述一次手动充值/调账请求。
type CreditParams struct {
	Amount  int64
	Reason  string
	RefType string
	RefID   string
}

// Credit 由 owner/admin 调用执行人工上分。
func (s *WalletService) Credit(ctx context.Context, p CreditParams) (domain.WalletTransaction, error) {
	return s.apply(ctx, p, domain.WalletKindCredit, +1)
}

// Refund 在某个外部任务被判失败时退还金额。
func (s *WalletService) Refund(ctx context.Context, p CreditParams) (domain.WalletTransaction, error) {
	return s.apply(ctx, p, domain.WalletKindRefund, +1)
}

// Debit 在外部任务实际消耗资源时扣费。
func (s *WalletService) Debit(ctx context.Context, p CreditParams) (domain.WalletTransaction, error) {
	return s.apply(ctx, p, domain.WalletKindDebit, -1)
}

// DebitOperation 根据操作类型自动扣费，支持幂等性。
// 若同一操作已扣费（根据 refType + refID），返回 ErrAlreadyDebited。
// 若操作类型未知，返回错误。
// 若余额不足，创建 pending_billing 记录，返回 ErrInsufficientBalance。
func (s *WalletService) DebitOperation(
	ctx context.Context,
	opType domain.OperationType,
	refType string,
	refID string,
) (domain.WalletTransaction, error) {
	auth, err := s.requireAuth(ctx)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	// 获取操作成本（先查数据库，回退到常量）
	cost, err := s.getOperationCostFromDB(ctx, auth.OrganizationID, opType)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	// 检查幂等性（是否已扣费）
	existingTx, err := s.repo.GetTransactionByRef(ctx, auth.OrganizationID, string(opType), refID)
	if err == nil && existingTx != nil {
		// 已存在相同 ref 的交易，返回已扣费
		return *existingTx, ErrAlreadyDebited
	}

	// 尝试扣费
	params := CreditParams{
		Amount:  cost,
		Reason:  fmt.Sprintf("operation: %s", opType),
		RefType: string(opType),
		RefID:   refID,
	}

	tx, err := s.apply(ctx, params, domain.WalletKindDebit, -1)
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			// 余额不足，创建待结算记录
			if s.pendingBillingRepo != nil {
				pb := &domain.PendingBilling{
					OrganizationID: auth.OrganizationID,
					OperationType:  opType,
					RefType:        string(opType),
					RefID:          refID,
					Amount:         cost,
					Status:         domain.PendingBillingStatusPending,
					MaxRetries:     5,
				}
				_ = s.pendingBillingRepo.Create(ctx, pb)
			}
		}
		return domain.WalletTransaction{}, err
	}

	return tx, nil
}

// DebitChatOperation 基于 token 数为对话操作扣费。
// 成本计算：(inputTokens + outputTokens) / 1000 * TokenCostPerThousand
// 参数：
// - inputTokens: 模型输入 tokens 数
// - outputTokens: 模型输出 tokens 数
// - refID: 对话消息 ID（用于幂等性检查）
func (s *WalletService) DebitChatOperation(
	ctx context.Context,
	inputTokens int64,
	outputTokens int64,
	refID string,
) (domain.WalletTransaction, error) {
	auth, err := s.requireAuth(ctx)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	// 计算成本
	cost := domain.CalculateChatCost(inputTokens, outputTokens)
	if cost == 0 {
		// 不产生成本的对话，不需要扣费
		return domain.WalletTransaction{}, nil
	}

	// 检查幂等性（是否已扣费）
	existingTx, err := s.repo.GetTransactionByRef(ctx, auth.OrganizationID, string(domain.OperationTypeChat), refID)
	if err == nil && existingTx != nil {
		// 已存在相同 ref 的交易，返回已扣费
		return *existingTx, ErrAlreadyDebited
	}

	// 尝试扣费
	params := CreditParams{
		Amount:  cost,
		Reason:  fmt.Sprintf("chat: %d input + %d output tokens", inputTokens, outputTokens),
		RefType: string(domain.OperationTypeChat),
		RefID:   refID,
	}

	tx, err := s.apply(ctx, params, domain.WalletKindDebit, -1)
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			// 余额不足，创建待结算记录
			if s.pendingBillingRepo != nil {
				pb := &domain.PendingBilling{
					OrganizationID: auth.OrganizationID,
					OperationType:  domain.OperationTypeChat,
					RefType:        string(domain.OperationTypeChat),
					RefID:          refID,
					Amount:         cost,
					Status:         domain.PendingBillingStatusPending,
					MaxRetries:     5,
				}
				_ = s.pendingBillingRepo.Create(ctx, pb)
			}
		}
		return domain.WalletTransaction{}, err
	}

	return tx, nil
}

func (s *WalletService) apply(
	ctx context.Context,
	p CreditParams,
	kind domain.WalletTransactionKind,
	direction int,
) (domain.WalletTransaction, error) {
	auth, err := s.requireAuth(ctx)
	if err != nil {
		return domain.WalletTransaction{}, err
	}
	if p.Amount <= 0 {
		return domain.WalletTransaction{}, errors.New("wallet: amount must be positive")
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.WalletTransaction{}, err
	}
	_, tx, err := s.repo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: auth.OrganizationID,
		Kind:           kind,
		Direction:      direction,
		Amount:         p.Amount,
		Reason:         strings.TrimSpace(p.Reason),
		RefType:        strings.TrimSpace(p.RefType),
		RefID:          strings.TrimSpace(p.RefID),
		ActorUserID:    auth.UserID,
		TransactionID:  id,
		CreatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	// Create notification for wallet events
	if s.notificationSvc != nil {
		notifKind := domain.NotificationKindWalletCredit
		title := "钱包充值"
		body := fmt.Sprintf("增加 %d 积分", p.Amount)

		if kind == domain.WalletKindDebit {
			notifKind = domain.NotificationKindWalletDebit
			title = "钱包扣费"
			body = fmt.Sprintf("扣除 %d 积分", p.Amount)
		}

		_, _ = s.notificationSvc.CreateNotification(ctx, auth.OrganizationID, notifKind, title, body, &auth.UserID, map[string]interface{}{
			"amount":         p.Amount,
			"reason":         p.Reason,
			"transaction_id": id,
			"balance_after":  tx.BalanceAfter,
		})
	}

	return tx, nil
}
