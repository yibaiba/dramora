package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider/payment"
	"github.com/yibaiba/dramora/internal/repo"
)

// PaymentService 支付服务
type PaymentService struct {
	paymentOrderRepo repo.PaymentOrderRepository
	walletService    *WalletService
	provider         payment.Provider
	logger           *slog.Logger
}

// NewPaymentService 创建支付服务
func NewPaymentService(
	paymentOrderRepo repo.PaymentOrderRepository,
	walletService *WalletService,
	provider payment.Provider,
	logger *slog.Logger,
) *PaymentService {
	return &PaymentService{
		paymentOrderRepo: paymentOrderRepo,
		walletService:    walletService,
		provider:         provider,
		logger:           logger,
	}
}

// InitiateCharge 初始化充值，返回支付网关 URL
func (s *PaymentService) InitiateCharge(ctx context.Context, userID, organizationID, orderID string, amount int64, currency string) (ChargeSession, error) {
	// 创建支付会话
	resp, err := s.provider.CreateSession(ctx, payment.CreateSessionRequest{
		Amount:      amount,
		Currency:    currency,
		UserID:      userID,
		OrderID:     orderID,
		Description: fmt.Sprintf("Dramora Credits - %d %s", amount/100, currency),
	})
	if err != nil {
		s.logger.Error("failed to create payment session", slog.String("error", err.Error()))
		return ChargeSession{}, fmt.Errorf("failed to create payment session: %w", err)
	}

	// 保存支付订单到数据库（初始状态：pending）
	order := &repo.PaymentOrder{
		ID:                orderID,
		UserID:            userID,
		OrganizationID:    organizationID,
		Provider:          "stripe",
		ProviderSessionID: resp.SessionID,
		Amount:            amount,
		Currency:          currency,
		Status:            "pending",
		CreatedAt:         time.Now(),
	}
	if err := s.paymentOrderRepo.Create(ctx, order); err != nil {
		s.logger.Error("failed to create payment order", slog.String("error", err.Error()))
		return ChargeSession{}, fmt.Errorf("failed to create payment order: %w", err)
	}

	return ChargeSession{
		SessionID: resp.SessionID,
		URL:       resp.URL,
		OrderID:   orderID,
	}, nil
}

// ProcessWebhook 处理支付网关回调，更新钱包
func (s *PaymentService) ProcessWebhook(ctx context.Context, payload payment.WebhookPayload) error {
	// 查询对应的支付订单
	order, err := s.paymentOrderRepo.GetByProviderSessionID(ctx, "stripe", payload.SessionID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("payment order not found for webhook", slog.String("sessionID", payload.SessionID))
			return nil // 已处理过的 webhook 或未知订单，不重复处理
		}
		return fmt.Errorf("failed to get payment order: %w", err)
	}

	// 为 webhook 处理注入系统认证上下文（确保有组织隔离）
	authCtx := WithRequestAuthContext(ctx, RequestAuthContext{
		UserID:         "system_payment",
		OrganizationID: order.OrganizationID,
		Role:           "system",
	})

	now := time.Now()
	if payload.Status == "success" {
		// 更新订单状态为成功
		if err := s.paymentOrderRepo.UpdateStatus(ctx, order.ID, "success", &now, nil); err != nil {
			s.logger.Error("failed to update payment order status", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update payment order: %w", err)
		}

		// 向钱包充值（使用注入了组织上下文的 authCtx）
		txn, err := s.walletService.Credit(authCtx, CreditParams{
			Amount:  payload.Amount,
			Reason:  fmt.Sprintf("Payment charge - Order %s", order.ID),
			RefType: "payment_order",
			RefID:   order.ID,
		})
		if err != nil {
			s.logger.Error("failed to credit wallet", slog.String("error", err.Error()))
			// 记录充值失败的原因
			errMsg := err.Error()
			_ = s.paymentOrderRepo.UpdateStatus(ctx, order.ID, "failed", nil, &errMsg)
			return fmt.Errorf("failed to credit wallet: %w", err)
		}

		// 关联钱包交易记录
		if err := s.paymentOrderRepo.UpdateWalletSnapshot(ctx, order.ID, txn.ID); err != nil {
			s.logger.Error("failed to update payment order wallet transaction", slog.String("error", err.Error()))
			// 这个错误不影响支付成功，仅记录
		}

		s.logger.Info("payment webhook processed successfully",
			slog.String("orderID", order.ID),
			slog.String("sessionID", payload.SessionID),
			slog.Int64("amount", payload.Amount),
		)
	} else {
		// 支付失败
		errorMsg := "payment failed"
		if err := s.paymentOrderRepo.UpdateStatus(ctx, order.ID, "failed", &now, &errorMsg); err != nil {
			s.logger.Error("failed to update payment order status", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update payment order: %w", err)
		}
		s.logger.Warn("payment webhook processed with failure status",
			slog.String("orderID", order.ID),
			slog.String("sessionID", payload.SessionID),
		)
	}

	return nil
}

// ChargeSession 充值会话信息
type ChargeSession struct {
	SessionID string // 支付会话 ID
	URL       string // 支付网关重定向 URL
	OrderID   string // 订单 ID
}

// Provider 返回底层支付提供商实例
func (s *PaymentService) Provider() payment.Provider {
	return s.provider
}
