package repo

import (
"context"
"time"
)

// PaymentOrder 支付订单记录
type PaymentOrder struct {
ID                string
UserID            string
OrganizationID    string
Provider          string    // "stripe" / "alipay" / "wechat"
ProviderSessionID string    // Stripe session_id
Amount            int64     // 金额（分）
Currency          string    // 货币
Status            string    // "pending" / "success" / "failed" / "cancelled"
ErrorReason       *string   // 错误原因
WalletSnapshotID  *string   // 关联的钱包快照 ID
CreatedAt         time.Time
CompletedAt       *time.Time
}

// PaymentOrderRepository 支付订单数据库操作
type PaymentOrderRepository interface {
// Create 创建新的支付订单
Create(ctx context.Context, order *PaymentOrder) error

// GetByID 获取支付订单
GetByID(ctx context.Context, id string) (*PaymentOrder, error)

// GetByProviderSessionID 通过支付网关 session ID 查询订单
GetByProviderSessionID(ctx context.Context, provider, sessionID string) (*PaymentOrder, error)

// UpdateStatus 更新订单状态
UpdateStatus(ctx context.Context, id string, status string, completedAt *time.Time, errorReason *string) error

// UpdateWalletSnapshot 关联钱包快照
UpdateWalletSnapshot(ctx context.Context, id string, walletSnapshotID string) error

// ListByUserID 查询用户的支付订单（分页）
ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*PaymentOrder, error)
}

// NewPaymentOrderRepository 创建支付订单 repository
// MVP 使用内存实现
func NewPaymentOrderRepository(db interface{}) PaymentOrderRepository {
return NewMemoryPaymentOrderRepository()
}
