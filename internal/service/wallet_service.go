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

// WalletService 围绕 WalletRepository 实现按组织上下文的钱包能力。
type WalletService struct {
	repo repo.WalletRepository
	notificationSvc *NotificationService
}

// NewWalletService 构造 WalletService；repo 为 nil 时所有方法均返回 ErrUnauthorized。
func NewWalletService(r repo.WalletRepository, notifSvc *NotificationService) *WalletService {
	return &WalletService{repo: r, notificationSvc: notifSvc}
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
			"amount":           p.Amount,
			"reason":           p.Reason,
			"transaction_id":   id,
			"balance_after":    tx.BalanceAfter,
		})
	}
	
	return tx, nil
}
