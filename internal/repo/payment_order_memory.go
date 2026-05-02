package repo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

// memoryPaymentOrderRepository 内存实现的支付订单 repository
type memoryPaymentOrderRepository struct {
	mu        sync.RWMutex
	orders    map[string]*PaymentOrder
	bySession map[string]*PaymentOrder // provider_session_id -> order
}

// NewMemoryPaymentOrderRepository 创建内存支付订单 repository
func NewMemoryPaymentOrderRepository() PaymentOrderRepository {
	return &memoryPaymentOrderRepository{
		orders:    make(map[string]*PaymentOrder),
		bySession: make(map[string]*PaymentOrder),
	}
}

func (r *memoryPaymentOrderRepository) Create(ctx context.Context, order *PaymentOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.ID]; exists {
		return fmt.Errorf("payment order already exists: %s", order.ID)
	}

	// 深拷贝
	copy := *order
	r.orders[order.ID] = &copy
	r.bySession[order.ProviderSessionID] = &copy
	return nil
}

func (r *memoryPaymentOrderRepository) GetByID(ctx context.Context, id string) (*PaymentOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.orders[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return order, nil
}

func (r *memoryPaymentOrderRepository) GetByProviderSessionID(ctx context.Context, provider, sessionID string) (*PaymentOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.bySession[sessionID]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return order, nil
}

func (r *memoryPaymentOrderRepository) UpdateStatus(ctx context.Context, id string, status string, completedAt *time.Time, errorReason *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, exists := r.orders[id]
	if !exists {
		return domain.ErrNotFound
	}

	order.Status = status
	order.CompletedAt = completedAt
	order.ErrorReason = errorReason
	return nil
}

func (r *memoryPaymentOrderRepository) UpdateWalletSnapshot(ctx context.Context, id string, walletSnapshotID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, exists := r.orders[id]
	if !exists {
		return domain.ErrNotFound
	}

	order.WalletSnapshotID = &walletSnapshotID
	return nil
}

func (r *memoryPaymentOrderRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*PaymentOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*PaymentOrder
	for _, order := range r.orders {
		if order.UserID == userID {
			result = append(result, order)
		}
	}

	// 按创建时间倒序排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// 分页
	if offset >= len(result) {
		return []*PaymentOrder{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}
