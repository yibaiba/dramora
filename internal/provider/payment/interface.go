package payment

import (
	"context"
	"net/http"
)

// Provider 是支付网关的抽象接口
type Provider interface {
	// CreateSession 创建支付会话，返回重定向 URL
	CreateSession(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error)

	// VerifyWebhook 验证 webhook 签名并解析 payload
	VerifyWebhook(r *http.Request) (WebhookPayload, error)
}

// CreateSessionRequest 支付会话创建请求
type CreateSessionRequest struct {
	Amount      int64  // 金额（分）
	Currency    string // 货币（USD/CNY）
	UserID      string
	OrderID     string // 唯一订单号
	Description string
}

// CreateSessionResponse 支付会话创建响应
type CreateSessionResponse struct {
	URL       string // 支付网关重定向地址
	SessionID string // 支付会话 ID，用于追踪
}

// WebhookPayload 支付网关回调 payload
type WebhookPayload struct {
	OrderID   string // 订单 ID（对应 CreateSessionRequest.OrderID）
	SessionID string // 支付会话 ID
	Status    string // "success" / "failed" / "cancelled"
	Amount    int64  // 实际支付金额（分）
	Timestamp int64  // Unix 时间戳
}
