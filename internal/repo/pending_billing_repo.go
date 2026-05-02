package repo

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yibaiba/dramora/internal/domain"
)

// ErrNotFound 表示记录未找到。
var ErrNotFound = errors.New("not found")

// PendingBillingRepository 定义待结算记录的仓库接口。
type PendingBillingRepository interface {
	// Create 创建新的待结算记录。
	Create(ctx context.Context, pb *domain.PendingBilling) error

	// GetByID 根据 ID 获取待结算记录。
	GetByID(ctx context.Context, id string) (*domain.PendingBilling, error)

	// GetPending 获取指定数量的待重试记录（用于 worker）。
	GetPending(ctx context.Context, limit int) ([]*domain.PendingBilling, error)

	// Update 更新待结算记录。
	Update(ctx context.Context, pb *domain.PendingBilling) error

	// GetByRef 根据 ref_type 和 ref_id 查询是否存在待结算记录。
	GetByRef(ctx context.Context, orgID, refType, refID string) (*domain.PendingBilling, error)
}

// MemoryPendingBillingRepository 内存实现（用于测试）。
type MemoryPendingBillingRepository struct {
	mu  sync.RWMutex
	pbs map[string]*domain.PendingBilling
}

func NewMemoryPendingBillingRepository() *MemoryPendingBillingRepository {
	return &MemoryPendingBillingRepository{
		pbs: make(map[string]*domain.PendingBilling),
	}
}

func (r *MemoryPendingBillingRepository) Create(ctx context.Context, pb *domain.PendingBilling) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pb.ID == "" {
		pb.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	pb.CreatedAt = now
	pb.UpdatedAt = now

	r.pbs[pb.ID] = pb
	return nil
}

func (r *MemoryPendingBillingRepository) GetByID(ctx context.Context, id string) (*domain.PendingBilling, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pb, ok := r.pbs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return pb, nil
}

func (r *MemoryPendingBillingRepository) GetPending(ctx context.Context, limit int) ([]*domain.PendingBilling, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pending []*domain.PendingBilling
	for _, pb := range r.pbs {
		if pb.Status == domain.PendingBillingStatusPending || pb.Status == domain.PendingBillingStatusRetrying {
			pending = append(pending, pb)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (r *MemoryPendingBillingRepository) Update(ctx context.Context, pb *domain.PendingBilling) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.pbs[pb.ID]; !ok {
		return ErrNotFound
	}
	pb.UpdatedAt = time.Now().Unix()
	r.pbs[pb.ID] = pb
	return nil
}

func (r *MemoryPendingBillingRepository) GetByRef(ctx context.Context, orgID, refType, refID string) (*domain.PendingBilling, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pb := range r.pbs {
		if pb.OrganizationID == orgID && pb.RefType == refType && pb.RefID == refID {
			return pb, nil
		}
	}
	return nil, ErrNotFound
}

// PostgresPendingBillingRepository PostgreSQL 实现。
type PostgresPendingBillingRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPendingBillingRepository(pool *pgxpool.Pool) *PostgresPendingBillingRepository {
	return &PostgresPendingBillingRepository{pool: pool}
}

func (r *PostgresPendingBillingRepository) Create(ctx context.Context, pb *domain.PendingBilling) error {
	if pb.ID == "" {
		pb.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	pb.CreatedAt = now
	pb.UpdatedAt = now

	const query = `
		INSERT INTO pending_billings 
		(id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, query,
		pb.ID, pb.OrganizationID, pb.OperationType, pb.RefType, pb.RefID,
		pb.Amount, pb.Status, pb.RetryCount, pb.MaxRetries, pb.LastErrorMsg,
		pb.CreatedAt, pb.UpdatedAt,
	)
	return err
}

func (r *PostgresPendingBillingRepository) GetByID(ctx context.Context, id string) (*domain.PendingBilling, error) {
	pb := &domain.PendingBilling{}
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE id = $1
	`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
		&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
		&pb.CreatedAt, &pb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return pb, err
}

func (r *PostgresPendingBillingRepository) GetPending(ctx context.Context, limit int) ([]*domain.PendingBilling, error) {
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE status IN ('pending', 'retrying')
		ORDER BY updated_at ASC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pbs []*domain.PendingBilling
	for rows.Next() {
		pb := &domain.PendingBilling{}
		err := rows.Scan(
			&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
			&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
			&pb.CreatedAt, &pb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		pbs = append(pbs, pb)
	}

	return pbs, rows.Err()
}

func (r *PostgresPendingBillingRepository) Update(ctx context.Context, pb *domain.PendingBilling) error {
	pb.UpdatedAt = time.Now().Unix()
	const query = `
		UPDATE pending_billings
		SET status = $1, retry_count = $2, last_error_msg = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query, pb.Status, pb.RetryCount, pb.LastErrorMsg, pb.UpdatedAt, pb.ID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresPendingBillingRepository) GetByRef(ctx context.Context, orgID, refType, refID string) (*domain.PendingBilling, error) {
	pb := &domain.PendingBilling{}
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE organization_id = $1 AND ref_type = $2 AND ref_id = $3
	`

	err := r.pool.QueryRow(ctx, query, orgID, refType, refID).Scan(
		&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
		&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
		&pb.CreatedAt, &pb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pb, nil
}

// SQLitePendingBillingRepository SQLite 实现（本地开发）。
type SQLitePendingBillingRepository struct {
	db *sql.DB
}

func NewSQLitePendingBillingRepository(db *sql.DB) *SQLitePendingBillingRepository {
	return &SQLitePendingBillingRepository{db: db}
}

func (r *SQLitePendingBillingRepository) Create(ctx context.Context, pb *domain.PendingBilling) error {
	if pb.ID == "" {
		pb.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	pb.CreatedAt = now
	pb.UpdatedAt = now

	const query = `
		INSERT INTO pending_billings 
		(id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		pb.ID, pb.OrganizationID, pb.OperationType, pb.RefType, pb.RefID,
		pb.Amount, pb.Status, pb.RetryCount, pb.MaxRetries, pb.LastErrorMsg,
		pb.CreatedAt, pb.UpdatedAt,
	)
	return err
}

func (r *SQLitePendingBillingRepository) GetByID(ctx context.Context, id string) (*domain.PendingBilling, error) {
	pb := &domain.PendingBilling{}
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE id = ?
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
		&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
		&pb.CreatedAt, &pb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return pb, err
}

func (r *SQLitePendingBillingRepository) GetPending(ctx context.Context, limit int) ([]*domain.PendingBilling, error) {
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE status IN ('pending', 'retrying')
		ORDER BY updated_at ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pbs []*domain.PendingBilling
	for rows.Next() {
		pb := &domain.PendingBilling{}
		err := rows.Scan(
			&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
			&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
			&pb.CreatedAt, &pb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		pbs = append(pbs, pb)
	}

	return pbs, rows.Err()
}

func (r *SQLitePendingBillingRepository) Update(ctx context.Context, pb *domain.PendingBilling) error {
	pb.UpdatedAt = time.Now().Unix()
	const query = `
		UPDATE pending_billings
		SET status = ?, retry_count = ?, last_error_msg = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, pb.Status, pb.RetryCount, pb.LastErrorMsg, pb.UpdatedAt, pb.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SQLitePendingBillingRepository) GetByRef(ctx context.Context, orgID, refType, refID string) (*domain.PendingBilling, error) {
	pb := &domain.PendingBilling{}
	const query = `
		SELECT id, organization_id, operation_type, ref_type, ref_id, amount, status, retry_count, max_retries, last_error_msg, created_at, updated_at
		FROM pending_billings
		WHERE organization_id = ? AND ref_type = ? AND ref_id = ?
	`

	err := r.db.QueryRowContext(ctx, query, orgID, refType, refID).Scan(
		&pb.ID, &pb.OrganizationID, &pb.OperationType, &pb.RefType, &pb.RefID,
		&pb.Amount, &pb.Status, &pb.RetryCount, &pb.MaxRetries, &pb.LastErrorMsg,
		&pb.CreatedAt, &pb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pb, nil
}
