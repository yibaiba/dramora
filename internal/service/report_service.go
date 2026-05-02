package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// ReportService 处理清算报表的生成与查询。
type ReportService struct {
	walletRepo         repo.WalletRepository
	pendingBillingRepo repo.PendingBillingRepository
	operationCostRepo  repo.OperationCostRepository
	reportRepo         repo.BillingReportRepository
}

// NewReportService 构造 ReportService。
func NewReportService(
	walletRepo repo.WalletRepository,
	pendingBillingRepo repo.PendingBillingRepository,
	operationCostRepo repo.OperationCostRepository,
	reportRepo repo.BillingReportRepository,
) *ReportService {
	return &ReportService{
		walletRepo:         walletRepo,
		pendingBillingRepo: pendingBillingRepo,
		operationCostRepo:  operationCostRepo,
		reportRepo:         reportRepo,
	}
}

// GenerateReport 生成清算报表。
// 统计指定期间内所有交易记录，按操作类型分类，包括待结算状态统计。
func (s *ReportService) GenerateReport(
	ctx context.Context,
	orgID string,
	periodStart, periodEnd int64,
	userID string,
) (*domain.BillingReport, error) {
	reportID := uuid.New().String()
	now := time.Now().Unix()

	// 1. 查询期间内所有交易（使用新的 List 方法进行日期范围过滤）
	startTime := time.Unix(periodStart, 0).UTC()
	endTime := time.Unix(periodEnd, 0).UTC()
	opts := repo.WalletTransactionFilterOptions{
		OrganizationID: orgID,
		StartTime:      startTime,
		EndTime:        endTime,
		Limit:          1000,
		Offset:         0,
	}
	page, err := s.walletRepo.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	// 2. 统计各类交易金额
	stats := make(map[domain.WalletTransactionKind]int64)
	operationStats := make(map[domain.OperationType]map[string]int64) // op_type -> {count, amount}

	for _, tx := range page.Transactions {
		// 统计总额
		if tx.Kind == domain.WalletKindAdjust {
			// adjust 根据 Direction 判断增减
			if tx.Direction > 0 {
				stats[domain.WalletKindCredit] += tx.Amount
			} else {
				stats[domain.WalletKindDebit] += tx.Amount
			}
		} else {
			stats[tx.Kind] += tx.Amount
		}

		// 统计按操作类型
		if tx.RefType != "" {
			// 尝试识别操作类型（从 RefType 提取或使用 RefID）
			opType := inferOperationTypeFromRefType(tx.RefType)
			if opType != "" {
				if _, ok := operationStats[opType]; !ok {
					operationStats[opType] = make(map[string]int64)
				}
				operationStats[opType]["count"]++
				operationStats[opType]["amount"] += tx.Amount
			}
		}
	}

	// 3. 查询待结算统计（使用新的 List 方法按时间范围和状态过滤）
	pendingCount := int(0)
	pendingAmount := int64(0)
	resolvedCount := int(0)
	failedCount := int(0)
	pbOpts := repo.PendingBillingFilterOptions{
		OrganizationID: orgID,
		StartTime:      startTime,
		EndTime:        endTime,
		Limit:          1000,
		Offset:         0,
	}
	pendingBillings, err := s.pendingBillingRepo.List(ctx, pbOpts)
	if err == nil {
		for _, pb := range pendingBillings {
			switch pb.Status {
			case domain.PendingBillingStatusPending, domain.PendingBillingStatusRetrying:
				pendingCount++
				pendingAmount += pb.Amount
			case domain.PendingBillingStatusResolved:
				resolvedCount++
			case domain.PendingBillingStatusFailed:
				failedCount++
			}
		}
	}

	// 4. 计算净额
	creditAmount := stats[domain.WalletKindCredit]
	refundAmount := stats[domain.WalletKindRefund]
	debitAmount := stats[domain.WalletKindDebit]
	adjustAmount := stats[domain.WalletKindAdjust]
	netAmount := creditAmount + refundAmount - debitAmount + adjustAmount

	// 5. 创建报表
	report := &domain.BillingReport{
		ID:                   reportID,
		OrganizationID:       orgID,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		TotalDebitAmount:     debitAmount,
		TotalCreditAmount:    creditAmount,
		TotalRefundAmount:    refundAmount,
		TotalAdjustAmount:    adjustAmount,
		NetAmount:            netAmount,
		PendingBillingCount:  pendingCount,
		PendingBillingAmount: pendingAmount,
		ResolvedBillingCount: resolvedCount,
		FailedBillingCount:   failedCount,
		Status:               domain.ReportStatusDraft,
		GeneratedAt:          now,
		GeneratedBy:          userID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// 6. 存储报表
	err = s.reportRepo.Create(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	// 7. 创建明细记录
	for opType, stats := range operationStats {
		// 获取该操作类型的单位成本
		costRow, err := s.operationCostRepo.GetCost(ctx, orgID, opType)
		if err != nil {
			// 成本查询失败，使用默认值
			costRow, _ = s.operationCostRepo.GetCost(ctx, orgID, opType)
		}
		unitCost := int64(0)
		if costRow != nil {
			unitCost = costRow.CreditsCost
		}

		breakdown := &domain.BillingBreakdown{
			OperationType:    opType,
			UnitCost:         unitCost,
			UsageCount:       stats["count"],
			TotalDebitAmount: stats["amount"],
		}
		_ = s.reportRepo.CreateBreakdown(ctx, reportID, breakdown)
	}

	return report, nil
}

// GetReport 获取报表详情。
func (s *ReportService) GetReport(ctx context.Context, reportID string) (*domain.BillingReport, error) {
	return s.reportRepo.GetByID(ctx, reportID)
}

// ListReports 列表查询报表。
func (s *ReportService) ListReports(ctx context.Context, orgID string, limit, offset int) ([]*domain.BillingReport, int, error) {
	return s.reportRepo.ListByOrg(ctx, orgID, limit, offset)
}

// GetReportBreakdowns 获取报表的明细。
func (s *ReportService) GetReportBreakdowns(ctx context.Context, reportID string) ([]*domain.BillingBreakdown, error) {
	return s.reportRepo.GetBreakdowns(ctx, reportID)
}

// FinalizeReport 将报表标记为已最终确认。
func (s *ReportService) FinalizeReport(ctx context.Context, reportID string) error {
	return s.reportRepo.UpdateStatus(ctx, reportID, domain.ReportStatusFinalized)
}

// inferOperationTypeFromRefType 根据 RefType 推断操作类型。
func inferOperationTypeFromRefType(refType string) domain.OperationType {
	switch refType {
	case "story_analysis_id":
		return domain.OperationTypeStoryAnalysis
	case "image_generation_id":
		return domain.OperationTypeImageGeneration
	case "video_generation_id":
		return domain.OperationTypeVideoGeneration
	case "chat_id":
		return domain.OperationTypeChat
	case "storyboard_edit_id":
		return domain.OperationTypeStoryboardEdit
	case "character_edit_id":
		return domain.OperationTypeCharacterEdit
	case "scene_edit_id":
		return domain.OperationTypeSceneEdit
	default:
		return ""
	}
}
