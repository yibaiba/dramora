package repo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yibaiba/dramora/internal/domain"
)

// OperationCostRepository 定义操作成本的仓库接口。
type OperationCostRepository interface {
	// GetCost 获取指定操作类型和组织的当前有效成本。
	// 返回 nil 如果未找到（应回退到常量）。
	GetCost(ctx context.Context, orgID string, opType domain.OperationType) (*domain.OperationCostRow, error)

	// GetAllCosts 获取所有有效的操作成本（用于前端显示）。
	GetAllCosts(ctx context.Context, orgID string) ([]*domain.OperationCostRow, error)

	// CreateCost 创建新的操作成本记录。
	CreateCost(ctx context.Context, cost *domain.OperationCostRow) error

	// UpdateCost 更新操作成本（新建记录，不修改现有记录）。
	UpdateCost(ctx context.Context, oldCost, newCost *domain.OperationCostRow, reason string, changedBy string) error

	// GetCostHistory 获取指定操作类型的修改历史。
	GetCostHistory(ctx context.Context, orgID string, opType domain.OperationType) ([]*domain.OperationCostHistoryRow, error)
}

// MemoryOperationCostRepository 内存实现（MVP 和测试）。
type MemoryOperationCostRepository struct {
	mu    sync.RWMutex
	costs map[string]*domain.OperationCostRow // key: org_id:op_type
}

func NewMemoryOperationCostRepository() *MemoryOperationCostRepository {
	repo := &MemoryOperationCostRepository{
		costs: make(map[string]*domain.OperationCostRow),
	}
	// 初始化默认成本
	defaultOrgID := "00000000-0000-0000-0000-000000000001"
	now := time.Now().Unix()
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeStoryAnalysis)] = &domain.OperationCostRow{
		ID:             1,
		OperationType:  domain.OperationTypeStoryAnalysis,
		OrganizationID: defaultOrgID,
		CreditsCost:    50,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeImageGeneration)] = &domain.OperationCostRow{
		ID:             2,
		OperationType:  domain.OperationTypeImageGeneration,
		OrganizationID: defaultOrgID,
		CreditsCost:    100,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeVideoGeneration)] = &domain.OperationCostRow{
		ID:             3,
		OperationType:  domain.OperationTypeVideoGeneration,
		OrganizationID: defaultOrgID,
		CreditsCost:    200,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeChat)] = &domain.OperationCostRow{
		ID:             4,
		OperationType:  domain.OperationTypeChat,
		OrganizationID: defaultOrgID,
		CreditsCost:    0, // Chat 使用 token 计费
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeStoryboardEdit)] = &domain.OperationCostRow{
		ID:             5,
		OperationType:  domain.OperationTypeStoryboardEdit,
		OrganizationID: defaultOrgID,
		CreditsCost:    5,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeCharacterEdit)] = &domain.OperationCostRow{
		ID:             6,
		OperationType:  domain.OperationTypeCharacterEdit,
		OrganizationID: defaultOrgID,
		CreditsCost:    5,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.costs[fmt.Sprintf("%s:%s", defaultOrgID, domain.OperationTypeSceneEdit)] = &domain.OperationCostRow{
		ID:             7,
		OperationType:  domain.OperationTypeSceneEdit,
		OrganizationID: defaultOrgID,
		CreditsCost:    5,
		EffectiveAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return repo
}

func (r *MemoryOperationCostRepository) GetCost(ctx context.Context, orgID string, opType domain.OperationType) (*domain.OperationCostRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", orgID, opType)
	if cost, ok := r.costs[key]; ok {
		return cost, nil
	}
	return nil, nil
}

func (r *MemoryOperationCostRepository) GetAllCosts(ctx context.Context, orgID string) ([]*domain.OperationCostRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var costs []*domain.OperationCostRow
	for _, cost := range r.costs {
		if cost.OrganizationID == orgID {
			costs = append(costs, cost)
		}
	}
	return costs, nil
}

func (r *MemoryOperationCostRepository) CreateCost(ctx context.Context, cost *domain.OperationCostRow) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", cost.OrganizationID, cost.OperationType)
	r.costs[key] = cost
	return nil
}

func (r *MemoryOperationCostRepository) UpdateCost(ctx context.Context, oldCost, newCost *domain.OperationCostRow, reason string, changedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", newCost.OrganizationID, newCost.OperationType)
	r.costs[key] = newCost
	// 内存实现不记录历史，但实际应记录
	return nil
}

func (r *MemoryOperationCostRepository) GetCostHistory(ctx context.Context, orgID string, opType domain.OperationType) ([]*domain.OperationCostHistoryRow, error) {
	// 内存实现不提供历史记录；PostgreSQL 实现会提供
	// 返回空列表而不是错误，避免中断 admin 查询流程
	return []*domain.OperationCostHistoryRow{}, nil
}

// PostgresOperationCostRepository PostgreSQL 实现。
type PostgresOperationCostRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOperationCostRepository(pool *pgxpool.Pool) *PostgresOperationCostRepository {
	return &PostgresOperationCostRepository{pool: pool}
}

func (r *PostgresOperationCostRepository) GetCost(ctx context.Context, orgID string, opType domain.OperationType) (*domain.OperationCostRow, error) {
	query := `
		SELECT 
			id, operation_type, organization_id, credits_cost, 
			effective_at, expires_at, notes, created_at, created_by, updated_at, updated_by
		FROM operation_costs
		WHERE operation_type = $1 
			AND organization_id = $2
			AND effective_at <= NOW()
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY effective_at DESC
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, opType, orgID)
	cost := &domain.OperationCostRow{}
	err := row.Scan(
		&cost.ID, &cost.OperationType, &cost.OrganizationID, &cost.CreditsCost,
		&cost.EffectiveAt, &cost.ExpiresAt, &cost.Notes, &cost.CreatedAt, &cost.CreatedBy, &cost.UpdatedAt, &cost.UpdatedBy,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return cost, nil
}

func (r *PostgresOperationCostRepository) GetAllCosts(ctx context.Context, orgID string) ([]*domain.OperationCostRow, error) {
	query := `
		SELECT 
			id, operation_type, organization_id, credits_cost, 
			effective_at, expires_at, notes, created_at, created_by, updated_at, updated_by
		FROM operation_costs
		WHERE organization_id = $1
			AND effective_at <= NOW()
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY operation_type, effective_at DESC
	`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var costs []*domain.OperationCostRow
	for rows.Next() {
		cost := &domain.OperationCostRow{}
		err := rows.Scan(
			&cost.ID, &cost.OperationType, &cost.OrganizationID, &cost.CreditsCost,
			&cost.EffectiveAt, &cost.ExpiresAt, &cost.Notes, &cost.CreatedAt, &cost.CreatedBy, &cost.UpdatedAt, &cost.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		costs = append(costs, cost)
	}
	return costs, rows.Err()
}

func (r *PostgresOperationCostRepository) CreateCost(ctx context.Context, cost *domain.OperationCostRow) error {
	query := `
		INSERT INTO operation_costs (
			operation_type, organization_id, credits_cost, 
			effective_at, expires_at, notes, created_at, created_by, updated_at, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	now := time.Now().Unix()
	err := r.pool.QueryRow(ctx, query,
		cost.OperationType, cost.OrganizationID, cost.CreditsCost,
		cost.EffectiveAt, cost.ExpiresAt, cost.Notes, now, cost.CreatedBy, now, cost.CreatedBy,
	).Scan(&cost.ID)
	return err
}

func (r *PostgresOperationCostRepository) UpdateCost(ctx context.Context, oldCost, newCost *domain.OperationCostRow, reason string, changedBy string) error {
	// 标记旧记录为已过期
	if oldCost != nil {
		expireQuery := `UPDATE operation_costs SET expires_at = NOW() WHERE id = $1`
		_, err := r.pool.Exec(ctx, expireQuery, oldCost.ID)
		if err != nil {
			return err
		}
	}

	// 创建新记录
	if err := r.CreateCost(ctx, newCost); err != nil {
		return err
	}

	// 记录历史
	historyQuery := `
		INSERT INTO operation_cost_history (
			operation_type, organization_id, old_cost, new_cost,
			effective_at, reason, changed_by, changed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	oldCostVal := int64(0)
	if oldCost != nil {
		oldCostVal = oldCost.CreditsCost
	}

	_, err := r.pool.Exec(ctx, historyQuery,
		newCost.OperationType, newCost.OrganizationID, oldCostVal, newCost.CreditsCost,
		newCost.EffectiveAt, reason, changedBy,
	)
	return err
}

func (r *PostgresOperationCostRepository) GetCostHistory(ctx context.Context, orgID string, opType domain.OperationType) ([]*domain.OperationCostHistoryRow, error) {
	query := `
		SELECT 
			id, operation_type, organization_id, old_cost, new_cost,
			effective_at, reason, changed_by, changed_at
		FROM operation_cost_history
		WHERE operation_type = $1 
			AND organization_id = $2
		ORDER BY changed_at DESC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query, opType, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*domain.OperationCostHistoryRow
	for rows.Next() {
		h := &domain.OperationCostHistoryRow{}
		err := rows.Scan(
			&h.ID, &h.OperationType, &h.OrganizationID, &h.OldCost, &h.NewCost,
			&h.EffectiveAt, &h.Reason, &h.ChangedBy, &h.ChangedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}
