package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

// ShortCodeRepository 定义短链接仓库接口
type ShortCodeRepository interface {
	// Create 创建新的短链接
	Create(ctx context.Context, shortCode *domain.ShortCode) error

	// GetByShortCode 根据短码查询
	GetByShortCode(ctx context.Context, shortCode string) (*domain.ShortCode, error)

	// GetByCode 根据赎回码查询
	GetByCode(ctx context.Context, code string) (*domain.ShortCode, error)
}

// ShortCodeRepositoryImpl 短链接仓库实现
type ShortCodeRepositoryImpl struct {
	pool *pgxpool.Pool
}

// NewShortCodeRepository 创建短链接仓库实例
func NewShortCodeRepository(pool *pgxpool.Pool) ShortCodeRepository {
	return &ShortCodeRepositoryImpl{pool: pool}
}

// Create 创建新的短链接
func (r *ShortCodeRepositoryImpl) Create(ctx context.Context, sc *domain.ShortCode) error {
	query := `
		INSERT INTO short_codes (code, short_code, organization_id, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		sc.Code,
		sc.ShortCode,
		sc.OrganizationID,
	).Scan(&sc.ID, &sc.CreatedAt)

	if err != nil {
		return err
	}

	return nil
}

// GetByShortCode 根据短码查询
func (r *ShortCodeRepositoryImpl) GetByShortCode(ctx context.Context, shortCode string) (*domain.ShortCode, error) {
	query := `
		SELECT id, code, short_code, organization_id, created_at
		FROM short_codes
		WHERE short_code = $1
		LIMIT 1
	`

	var sc domain.ShortCode
	err := r.pool.QueryRow(ctx, query, shortCode).Scan(
		&sc.ID,
		&sc.Code,
		&sc.ShortCode,
		&sc.OrganizationID,
		&sc.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &sc, nil
}

// GetByCode 根据赎回码查询
func (r *ShortCodeRepositoryImpl) GetByCode(ctx context.Context, code string) (*domain.ShortCode, error) {
	query := `
		SELECT id, code, short_code, organization_id, created_at
		FROM short_codes
		WHERE code = $1
		LIMIT 1
	`

	var sc domain.ShortCode
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&sc.ID,
		&sc.Code,
		&sc.ShortCode,
		&sc.OrganizationID,
		&sc.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &sc, nil
}
