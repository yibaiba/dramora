package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

// billingReportResponse 清算报表 DTO。
type billingReportResponse struct {
	ID                   string `json:"id"`
	OrganizationID       string `json:"organization_id"`
	PeriodStart          int64  `json:"period_start"`
	PeriodEnd            int64  `json:"period_end"`
	TotalDebitAmount     int64  `json:"total_debit_amount"`
	TotalCreditAmount    int64  `json:"total_credit_amount"`
	TotalRefundAmount    int64  `json:"total_refund_amount"`
	TotalAdjustAmount    int64  `json:"total_adjust_amount"`
	NetAmount            int64  `json:"net_amount"`
	PendingBillingCount  int    `json:"pending_billing_count"`
	PendingBillingAmount int64  `json:"pending_billing_amount"`
	ResolvedBillingCount int    `json:"resolved_billing_count"`
	FailedBillingCount   int    `json:"failed_billing_count"`
	Status               string `json:"status"`
	GeneratedAt          int64  `json:"generated_at"`
	GeneratedBy          string `json:"generated_by"`
	CreatedAt            int64  `json:"created_at"`
}

func billingReportDTO(report *domain.BillingReport) billingReportResponse {
	return billingReportResponse{
		ID:                   report.ID,
		OrganizationID:       report.OrganizationID,
		PeriodStart:          report.PeriodStart,
		PeriodEnd:            report.PeriodEnd,
		TotalDebitAmount:     report.TotalDebitAmount,
		TotalCreditAmount:    report.TotalCreditAmount,
		TotalRefundAmount:    report.TotalRefundAmount,
		TotalAdjustAmount:    report.TotalAdjustAmount,
		NetAmount:            report.NetAmount,
		PendingBillingCount:  report.PendingBillingCount,
		PendingBillingAmount: report.PendingBillingAmount,
		ResolvedBillingCount: report.ResolvedBillingCount,
		FailedBillingCount:   report.FailedBillingCount,
		Status:               string(report.Status),
		GeneratedAt:          report.GeneratedAt,
		GeneratedBy:          report.GeneratedBy,
		CreatedAt:            report.CreatedAt,
	}
}

// billingBreakdownResponse 成本明细 DTO。
type billingBreakdownResponse struct {
	OperationType    string `json:"operation_type"`
	UnitCost         int64  `json:"unit_cost"`
	UsageCount       int64  `json:"usage_count"`
	TotalDebitAmount int64  `json:"total_debit_amount"`
}

func billingBreakdownDTO(breakdown *domain.BillingBreakdown) billingBreakdownResponse {
	return billingBreakdownResponse{
		OperationType:    string(breakdown.OperationType),
		UnitCost:         breakdown.UnitCost,
		UsageCount:       breakdown.UsageCount,
		TotalDebitAmount: breakdown.TotalDebitAmount,
	}
}

// billingReportDetailResponse 报表详情（包含明细）。
type billingReportDetailResponse struct {
	Report     billingReportResponse      `json:"report"`
	Breakdowns []billingBreakdownResponse `json:"breakdowns"`
}

// generateBillingReportRequest 生成报表请求。
type generateBillingReportRequest struct {
	PeriodStart int64 `json:"period_start"` // Unix timestamp
	PeriodEnd   int64 `json:"period_end"`   // Unix timestamp
}

// getAdminBillingReports 列表查询报表。
func (a *api) getAdminBillingReports(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	if a.reportService == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "report service not configured")
		return
	}

	// 解析分页参数
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	reports, total, err := a.reportService.ListReports(r.Context(), auth.OrganizationID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_failed", "failed to list reports")
		return
	}

	var dtos []billingReportResponse
	for _, report := range reports {
		dtos = append(dtos, billingReportDTO(report))
	}

	response := map[string]any{
		"reports": dtos,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}
	writeJSON(w, http.StatusOK, response)
}

// generateAdminBillingReport 生成新报表。
func (a *api) generateAdminBillingReport(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	if a.reportService == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "report service not configured")
		return
	}

	var req generateBillingReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
		return
	}

	// 验证日期范围
	if req.PeriodStart >= req.PeriodEnd {
		writeError(w, http.StatusBadRequest, "invalid_period", "period_start must be before period_end")
		return
	}

	// 限制查询范围（防止过大查询）
	maxSpan := int64(90 * 24 * 3600) // 90 天
	if req.PeriodEnd-req.PeriodStart > maxSpan {
		writeError(w, http.StatusBadRequest, "period_too_large", "maximum period span is 90 days")
		return
	}

	report, err := a.reportService.GenerateReport(r.Context(), auth.OrganizationID, req.PeriodStart, req.PeriodEnd, auth.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generation_failed", "failed to generate report")
		return
	}

	writeJSON(w, http.StatusCreated, billingReportDTO(report))
}

// getAdminBillingReportByID 查看单个报表详情。
func (a *api) getAdminBillingReportByID(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	if a.reportService == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "report service not configured")
		return
	}

	reportID := chi.URLParam(r, "reportID")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "report_id required")
		return
	}

	report, err := a.reportService.GetReport(r.Context(), reportID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "report not found")
		} else {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to get report")
		}
		return
	}

	// 验证组织隔离
	if report.OrganizationID != auth.OrganizationID {
		writeError(w, http.StatusForbidden, "insufficient_role", "cannot access report from another organization")
		return
	}

	breakdowns, _ := a.reportService.GetReportBreakdowns(r.Context(), reportID)
	var breakdownDTOs []billingBreakdownResponse
	if breakdowns != nil {
		for _, bd := range breakdowns {
			breakdownDTOs = append(breakdownDTOs, billingBreakdownDTO(bd))
		}
	}

	detail := billingReportDetailResponse{
		Report:     billingReportDTO(report),
		Breakdowns: breakdownDTOs,
	}

	writeJSON(w, http.StatusOK, detail)
}

// getAdminBillingReportSummary 获取报表摘要（轻量级响应）。
func (a *api) getAdminBillingReportSummary(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	if a.reportService == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "report service not configured")
		return
	}

	reportID := chi.URLParam(r, "reportID")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "report_id required")
		return
	}

	report, err := a.reportService.GetReport(r.Context(), reportID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "report not found")
		} else {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to get report")
		}
		return
	}

	// 验证组织隔离
	if report.OrganizationID != auth.OrganizationID {
		writeError(w, http.StatusForbidden, "insufficient_role", "cannot access report from another organization")
		return
	}

	// 返回摘要
	summary := map[string]any{
		"id":                     report.ID,
		"period_start":           report.PeriodStart,
		"period_end":             report.PeriodEnd,
		"total_debit_amount":     report.TotalDebitAmount,
		"total_credit_amount":    report.TotalCreditAmount,
		"total_refund_amount":    report.TotalRefundAmount,
		"net_amount":             report.NetAmount,
		"pending_billing_count":  report.PendingBillingCount,
		"resolved_billing_count": report.ResolvedBillingCount,
		"failed_billing_count":   report.FailedBillingCount,
		"status":                 report.Status,
		"generated_at":           report.GeneratedAt,
	}

	writeJSON(w, http.StatusOK, summary)
}
