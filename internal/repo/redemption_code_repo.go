package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

// RedemptionCodeRepository 定义赎回码仓库接口
type RedemptionCodeRepository interface {
	// CreateCampaign 创建新的赎回活动
	CreateCampaign(ctx context.Context, campaign *domain.RedemptionCampaign) error

	// GetCampaignByID 根据 ID 获取赎回活动
	GetCampaignByID(ctx context.Context, campaignID string) (*domain.RedemptionCampaign, error)

	// GenerateCodes 批量生成赎回码
	GenerateCodes(ctx context.Context, codes []*domain.RedemptionCode) error

	// GetCodeByCode 根据赎回码字符串查询
	GetCodeByCode(ctx context.Context, code string) (*domain.RedemptionCode, error)

	// RedeemCode 使用乐观锁完成兑换（原子操作）
	// 返回兑换后的赎回码，如果因版本冲突失败则返回 ErrCodeVersionConflict
	RedeemCode(ctx context.Context, code string, usedBy string, currentVersion int) (*domain.RedemptionCode, error)

	// GetCampaignStats 获取活动的统计数据
	GetCampaignStats(ctx context.Context, campaignID string) (*domain.CampaignStats, error)

	// ListCodes 按筛选条件列表赎回码
	ListCodes(ctx context.Context, filter RedemptionCodeFilter) ([]*domain.RedemptionCode, error)

	// ListCodesByCampaign 按活动 ID 列表赎回码
	ListCodesByCampaign(ctx context.Context, campaignID string, status string) ([]*domain.RedemptionCode, error)
}

// RedemptionCodeFilter 用于查询赎回码的筛选条件
type RedemptionCodeFilter struct {
	OrganizationID string
	Status         string // optional: "unused", "used", ""（全部）
	Limit          int
	Offset         int
}

// PostgresRedemptionCodeRepository 使用 PostgreSQL 的实现
type PostgresRedemptionCodeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRedemptionCodeRepository 创建 PostgreSQL 实现的仓库
func NewPostgresRedemptionCodeRepository(pool *pgxpool.Pool) *PostgresRedemptionCodeRepository {
	return &PostgresRedemptionCodeRepository{pool: pool}
}

// CreateCampaign 创建新的赎回活动
func (r *PostgresRedemptionCodeRepository) CreateCampaign(ctx context.Context, campaign *domain.RedemptionCampaign) error {
	if err := campaign.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO redemption_campaigns (id, name, description, status, created_by, organization_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.Description,
		string(campaign.Status),
		campaign.CreatedBy,
		campaign.OrganizationID,
		campaign.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}

	return nil
}

// scanRedemptionCode 从数据库行扫描赎回码，统一处理 Scan 逻辑
func (r *PostgresRedemptionCodeRepository) scanRedemptionCode(row pgx.Row) (*domain.RedemptionCode, error) {
	var rc domain.RedemptionCode
	var usedBy, campaignID, reason *string
	var usedAt, expiresAt *time.Time

	err := row.Scan(
		&rc.ID,
		&rc.Code,
		&rc.OrganizationID,
		&campaignID,
		&rc.Amount,
		&rc.Status,
		&rc.CreatedBy,
		&rc.CreatedAt,
		&usedBy,
		&usedAt,
		&expiresAt,
		&reason,
		&rc.Version,
	)

	if err != nil {
		return nil, err
	}

	// 统一处理 nullable 字段
	if usedBy != nil {
		rc.UsedBy = *usedBy
	}
	if usedAt != nil {
		rc.UsedAt = *usedAt
	}
	if expiresAt != nil {
		rc.ExpiresAt = *expiresAt
	}
	if campaignID != nil {
		rc.CampaignID = *campaignID
	}
	if reason != nil {
		rc.Reason = *reason
	}

	return &rc, nil
}

// GetCampaignByID 根据 ID 获取赎回活动
func (r *PostgresRedemptionCodeRepository) GetCampaignByID(ctx context.Context, campaignID string) (*domain.RedemptionCampaign, error) {
	query := `
		SELECT id, name, description, status, created_by, organization_id, created_at
		FROM redemption_campaigns
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, campaignID)

	var campaign domain.RedemptionCampaign
	err := row.Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.Description,
		&campaign.Status,
		&campaign.CreatedBy,
		&campaign.OrganizationID,
		&campaign.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, fmt.Errorf("get campaign: %w", err)
	}

	return &campaign, nil
}

// GenerateCodes 批量生成赎回码
func (r *PostgresRedemptionCodeRepository) GenerateCodes(ctx context.Context, codes []*domain.RedemptionCode) error {
	if len(codes) == 0 {
		return nil
	}

	// 验证所有码
	for _, code := range codes {
		if err := code.Validate(); err != nil {
			return err
		}
	}

	// 使用批量插入
	batch := &pgx.Batch{}

	query := `
		INSERT INTO redemption_codes 
		(id, code, organization_id, campaign_id, amount, status, created_by, created_at, expires_at, reason, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, code := range codes {
		batch.Queue(query,
			code.ID,
			strings.ToUpper(code.Code), // 存储为大写
			code.OrganizationID,
			code.CampaignID,
			code.Amount,
			string(code.Status),
			code.CreatedBy,
			code.CreatedAt,
			nullableTime(code.ExpiresAt),
			code.Reason,
			code.Version,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	// 执行所有插入
	for i := 0; i < len(codes); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("generate codes batch: %w", err)
		}
	}

	return nil
}

// GetCodeByCode 根据赎回码字符串查询
func (r *PostgresRedemptionCodeRepository) GetCodeByCode(ctx context.Context, code string) (*domain.RedemptionCode, error) {
	query := `
		SELECT id, code, organization_id, campaign_id, amount, status, created_by, created_at, 
		       used_by, used_at, expires_at, reason, version
		FROM redemption_codes
		WHERE UPPER(code) = UPPER($1)
	`

	row := r.pool.QueryRow(ctx, query, code)

	rc, err := r.scanRedemptionCode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCodeNotFound
		}
		return nil, fmt.Errorf("get code: %w", err)
	}

	return rc, nil
}

// RedeemCode 使用乐观锁完成兑换（原子操作）
func (r *PostgresRedemptionCodeRepository) RedeemCode(ctx context.Context, code string, usedBy string, currentVersion int) (*domain.RedemptionCode, error) {
	query := `
		UPDATE redemption_codes
		SET used_by = $1, used_at = NOW(), status = $2, version = version + 1
		WHERE UPPER(code) = UPPER($3) AND status = $4 AND version = $5
		RETURNING id, code, organization_id, campaign_id, amount, status, created_by, created_at,
		          used_by, used_at, expires_at, reason, version
	`

	row := r.pool.QueryRow(ctx, query,
		usedBy,
		string(domain.RedemptionCodeStatusUsed),
		code,
		string(domain.RedemptionCodeStatusUnused),
		currentVersion,
	)

	rc, err := r.scanRedemptionCode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 确定失败原因：已使用、版本冲突、或码不存在
			existing, err2 := r.GetCodeByCode(ctx, code)
			if err2 != nil {
				if errors.Is(err2, domain.ErrCodeNotFound) {
					return nil, domain.ErrCodeNotFound
				}
				return nil, err2
			}

			// 码存在，检查状态
			if existing.Status == domain.RedemptionCodeStatusUsed {
				return nil, domain.ErrCodeAlreadyUsed
			}

			// 检查过期
			if existing.IsExpired() {
				return nil, domain.ErrCodeExpired
			}

			// 版本冲突
			return nil, domain.ErrCodeVersionConflict
		}
		return nil, fmt.Errorf("redeem code: %w", err)
	}

	return rc, nil
}

// GetCampaignStats 获取活动的统计数据
func (r *PostgresRedemptionCodeRepository) GetCampaignStats(ctx context.Context, campaignID string) (*domain.CampaignStats, error) {
	query := `
		SELECT 
			c.id,
			c.name,
			COUNT(*) as total_codes,
			SUM(CASE WHEN rc.status = 'used' THEN 1 ELSE 0 END) as used_codes,
			SUM(CASE WHEN rc.status = 'unused' THEN 1 ELSE 0 END) as unused_codes,
			SUM(rc.amount) as total_amount,
			SUM(CASE WHEN rc.status = 'used' THEN rc.amount ELSE 0 END) as used_amount,
			SUM(CASE WHEN rc.status = 'unused' THEN rc.amount ELSE 0 END) as unused_amount
		FROM redemption_campaigns c
		LEFT JOIN redemption_codes rc ON c.id = rc.campaign_id
		WHERE c.id = $1
		GROUP BY c.id, c.name
	`

	row := r.pool.QueryRow(ctx, query, campaignID)

	var stats domain.CampaignStats
	var totalCodes, usedCodes, unusedCodes int64
	var totalAmount, usedAmount, unusedAmount *int64

	err := row.Scan(
		&stats.CampaignID,
		&stats.Name,
		&totalCodes,
		&usedCodes,
		&unusedCodes,
		&totalAmount,
		&usedAmount,
		&unusedAmount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, fmt.Errorf("get campaign stats: %w", err)
	}

	stats.TotalCodes = totalCodes
	stats.UsedCodes = usedCodes
	stats.UnusedCodes = unusedCodes

	if totalAmount != nil {
		stats.TotalAmount = *totalAmount
	}
	if usedAmount != nil {
		stats.UsedAmount = *usedAmount
	}
	if unusedAmount != nil {
		stats.UnusedAmount = *unusedAmount
	}

	if stats.TotalCodes > 0 {
		stats.UsagePercentage = float64(stats.UsedCodes) / float64(stats.TotalCodes) * 100
	}

	return &stats, nil
}

// ListCodes 按筛选条件列表赎回码
func (r *PostgresRedemptionCodeRepository) ListCodes(ctx context.Context, filter RedemptionCodeFilter) ([]*domain.RedemptionCode, error) {
	query := `
		SELECT id, code, organization_id, campaign_id, amount, status, created_by, created_at,
		       used_by, used_at, expires_at, reason, version
		FROM redemption_codes
		WHERE organization_id = $1
	`
	args := []interface{}{filter.OrganizationID}
	argCount := 2

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list codes: %w", err)
	}
	defer rows.Close()

	var codes []*domain.RedemptionCode
	for rows.Next() {
		rc, err := r.scanRedemptionCode(rows)
		if err != nil {
			return nil, fmt.Errorf("scan code: %w", err)
		}
		codes = append(codes, rc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list codes iteration: %w", err)
	}

	return codes, nil
}

// ListCodesByCampaign 按活动 ID 列表赎回码
func (r *PostgresRedemptionCodeRepository) ListCodesByCampaign(ctx context.Context, campaignID string, status string) ([]*domain.RedemptionCode, error) {
	query := `
		SELECT id, code, organization_id, campaign_id, amount, status, created_by, created_at,
		       used_by, used_at, expires_at, reason, version
		FROM redemption_codes
		WHERE campaign_id = $1
	`
	args := []interface{}{campaignID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list codes by campaign: %w", err)
	}
	defer rows.Close()

	var codes []*domain.RedemptionCode
	for rows.Next() {
		rc, err := r.scanRedemptionCode(rows)
		if err != nil {
			return nil, fmt.Errorf("scan code: %w", err)
		}
		codes = append(codes, rc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list codes by campaign iteration: %w", err)
	}

	return codes, nil
}

// nullableTime 将 time.Time 转换为 *time.Time（处理零值）
func nullableTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
