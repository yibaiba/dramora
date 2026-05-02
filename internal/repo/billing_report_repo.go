package repo

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yibaiba/dramora/internal/domain"
)

// BillingReportRepository 定义清算报表的仓库接口。
type BillingReportRepository interface {
	// Create 创建新报表。
	Create(ctx context.Context, report *domain.BillingReport) error

	// GetByID 按 ID 查询。
	GetByID(ctx context.Context, reportID string) (*domain.BillingReport, error)

	// ListByOrg 按组织列表报表。
	ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]*domain.BillingReport, int, error)

	// UpdateStatus 更新报表状态。
	UpdateStatus(ctx context.Context, reportID string, status domain.ReportStatus) error

	// GetBreakdowns 获取报表的操作类型明细。
	GetBreakdowns(ctx context.Context, reportID string) ([]*domain.BillingBreakdown, error)

	// CreateBreakdown 为报表创建明细记录。
	CreateBreakdown(ctx context.Context, reportID string, breakdown *domain.BillingBreakdown) error
}

// MemoryBillingReportRepository 内存实现（MVP）。
type MemoryBillingReportRepository struct {
	mu         sync.RWMutex
	reports    map[string]*domain.BillingReport
	breakdowns map[string][]*domain.BillingBreakdown // key: report_id
}

func NewMemoryBillingReportRepository() *MemoryBillingReportRepository {
	return &MemoryBillingReportRepository{
		reports:    make(map[string]*domain.BillingReport),
		breakdowns: make(map[string][]*domain.BillingBreakdown),
	}
}

func (r *MemoryBillingReportRepository) Create(ctx context.Context, report *domain.BillingReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[report.ID] = report
	return nil
}

func (r *MemoryBillingReportRepository) GetByID(ctx context.Context, reportID string) (*domain.BillingReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	report, ok := r.reports[reportID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return report, nil
}

func (r *MemoryBillingReportRepository) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]*domain.BillingReport, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var reports []*domain.BillingReport
	for _, report := range r.reports {
		if report.OrganizationID == orgID {
			reports = append(reports, report)
		}
	}
	// 简单分页实现
	total := len(reports)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return reports[start:end], total, nil
}

func (r *MemoryBillingReportRepository) UpdateStatus(ctx context.Context, reportID string, status domain.ReportStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	report, ok := r.reports[reportID]
	if !ok {
		return domain.ErrNotFound
	}
	report.Status = status
	report.UpdatedAt = time.Now().Unix()
	return nil
}

func (r *MemoryBillingReportRepository) GetBreakdowns(ctx context.Context, reportID string) ([]*domain.BillingBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	breakdowns, ok := r.breakdowns[reportID]
	if !ok {
		return []*domain.BillingBreakdown{}, nil
	}
	return breakdowns, nil
}

func (r *MemoryBillingReportRepository) CreateBreakdown(ctx context.Context, reportID string, breakdown *domain.BillingBreakdown) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.breakdowns[reportID] = append(r.breakdowns[reportID], breakdown)
	return nil
}

// PostgresBillingReportRepository PostgreSQL 实现。
type PostgresBillingReportRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBillingReportRepository(pool *pgxpool.Pool) *PostgresBillingReportRepository {
	return &PostgresBillingReportRepository{pool: pool}
}

func (r *PostgresBillingReportRepository) Create(ctx context.Context, report *domain.BillingReport) error {
	query := `
		INSERT INTO billing_reports (
			id, organization_id, period_start, period_end,
			total_debit_amount, total_credit_amount, total_refund_amount, total_adjust_amount,
			net_amount, pending_billing_count, pending_billing_amount,
			resolved_billing_count, failed_billing_count,
			status, generated_at, generated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`
	_, err := r.pool.Exec(ctx, query,
		report.ID,
		report.OrganizationID,
		report.PeriodStart,
		report.PeriodEnd,
		report.TotalDebitAmount,
		report.TotalCreditAmount,
		report.TotalRefundAmount,
		report.TotalAdjustAmount,
		report.NetAmount,
		report.PendingBillingCount,
		report.PendingBillingAmount,
		report.ResolvedBillingCount,
		report.FailedBillingCount,
		report.Status,
		report.GeneratedAt,
		report.GeneratedBy,
	)
	return err
}

func (r *PostgresBillingReportRepository) GetByID(ctx context.Context, reportID string) (*domain.BillingReport, error) {
	query := `
		SELECT 
			id, organization_id, period_start, period_end,
			total_debit_amount, total_credit_amount, total_refund_amount, total_adjust_amount,
			net_amount, pending_billing_count, pending_billing_amount,
			resolved_billing_count, failed_billing_count,
			status, generated_at, generated_by,
			EXTRACT(EPOCH FROM created_at)::BIGINT as created_at,
			EXTRACT(EPOCH FROM updated_at)::BIGINT as updated_at
		FROM billing_reports
		WHERE id = $1
	`
	var report domain.BillingReport
	err := r.pool.QueryRow(ctx, query, reportID).Scan(
		&report.ID, &report.OrganizationID, &report.PeriodStart, &report.PeriodEnd,
		&report.TotalDebitAmount, &report.TotalCreditAmount, &report.TotalRefundAmount, &report.TotalAdjustAmount,
		&report.NetAmount, &report.PendingBillingCount, &report.PendingBillingAmount,
		&report.ResolvedBillingCount, &report.FailedBillingCount,
		&report.Status, &report.GeneratedAt, &report.GeneratedBy,
		&report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &report, nil
}

func (r *PostgresBillingReportRepository) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]*domain.BillingReport, int, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM billing_reports WHERE organization_id = $1`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	query := `
		SELECT 
			id, organization_id, period_start, period_end,
			total_debit_amount, total_credit_amount, total_refund_amount, total_adjust_amount,
			net_amount, pending_billing_count, pending_billing_amount,
			resolved_billing_count, failed_billing_count,
			status, generated_at, generated_by,
			EXTRACT(EPOCH FROM created_at)::BIGINT as created_at,
			EXTRACT(EPOCH FROM updated_at)::BIGINT as updated_at
		FROM billing_reports
		WHERE organization_id = $1
		ORDER BY period_end DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []*domain.BillingReport
	for rows.Next() {
		var report domain.BillingReport
		err := rows.Scan(
			&report.ID, &report.OrganizationID, &report.PeriodStart, &report.PeriodEnd,
			&report.TotalDebitAmount, &report.TotalCreditAmount, &report.TotalRefundAmount, &report.TotalAdjustAmount,
			&report.NetAmount, &report.PendingBillingCount, &report.PendingBillingAmount,
			&report.ResolvedBillingCount, &report.FailedBillingCount,
			&report.Status, &report.GeneratedAt, &report.GeneratedBy,
			&report.CreatedAt, &report.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, &report)
	}
	return reports, total, rows.Err()
}

func (r *PostgresBillingReportRepository) UpdateStatus(ctx context.Context, reportID string, status domain.ReportStatus) error {
	query := `UPDATE billing_reports SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	result, err := r.pool.Exec(ctx, query, status, reportID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresBillingReportRepository) GetBreakdowns(ctx context.Context, reportID string) ([]*domain.BillingBreakdown, error) {
	query := `
		SELECT operation_type, unit_cost, usage_count, total_debit_amount
		FROM billing_report_breakdowns
		WHERE report_id = $1
		ORDER BY operation_type
	`
	rows, err := r.pool.Query(ctx, query, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdowns []*domain.BillingBreakdown
	for rows.Next() {
		var breakdown domain.BillingBreakdown
		var opType string
		err := rows.Scan(&opType, &breakdown.UnitCost, &breakdown.UsageCount, &breakdown.TotalDebitAmount)
		if err != nil {
			return nil, err
		}
		breakdown.OperationType = domain.OperationType(opType)
		breakdowns = append(breakdowns, &breakdown)
	}
	return breakdowns, rows.Err()
}

func (r *PostgresBillingReportRepository) CreateBreakdown(ctx context.Context, reportID string, breakdown *domain.BillingBreakdown) error {
	query := `
		INSERT INTO billing_report_breakdowns (
			report_id, operation_type, unit_cost, usage_count, total_debit_amount
		) VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query,
		reportID,
		string(breakdown.OperationType),
		breakdown.UnitCost,
		breakdown.UsageCount,
		breakdown.TotalDebitAmount,
	)
	return err
}
