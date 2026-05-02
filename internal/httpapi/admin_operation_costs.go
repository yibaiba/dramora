package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

// operationCostAdminResponse 对应单条操作成本记录的 DTO。
type operationCostAdminResponse struct {
	ID             int32     `json:"id"`
	OperationType  string    `json:"operation_type"`
	OrganizationID string    `json:"organization_id"`
	CreditsCost    int64     `json:"credits_cost"`
	EffectiveAt    int64     `json:"effective_at"`
	UpdatedAt      int64     `json:"updated_at"`
}

func operationCostAdminDTO(row *domain.OperationCostRow) operationCostAdminResponse {
	return operationCostAdminResponse{
		ID:             int32(row.ID),
		OperationType:  string(row.OperationType),
		OrganizationID: row.OrganizationID,
		CreditsCost:    row.CreditsCost,
		EffectiveAt:    row.EffectiveAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

// updateOperationCostRequest 更新请求体格式。
type updateOperationCostRequest struct {
	OperationType string `json:"operation_type"`
	CreditsCost   int64  `json:"credits_cost"`
}

// operationCostHistoryResponse 对应单条审计日志记录的 DTO。
type operationCostHistoryResponse struct {
	ID             int32     `json:"id"`
	OperationType  string    `json:"operation_type"`
	OrganizationID string    `json:"organization_id"`
	OldCost        *int64    `json:"old_cost"`
	NewCost        int64     `json:"new_cost"`
	EffectiveAt    int64     `json:"effective_at"`
	Reason         *string   `json:"reason"`
	ChangedBy      string    `json:"changed_by"`
	ChangedAt      int64     `json:"changed_at"`
}

func operationCostHistoryDTO(row *domain.OperationCostHistoryRow) operationCostHistoryResponse {
	return operationCostHistoryResponse{
		ID:             row.ID,
		OperationType:  string(row.OperationType),
		OrganizationID: row.OrganizationID,
		OldCost:        row.OldCost,
		NewCost:        row.NewCost,
		EffectiveAt:    row.EffectiveAt,
		Reason:         row.Reason,
		ChangedBy:      row.ChangedBy,
		ChangedAt:      row.ChangedAt,
	}
}

// GetAdminOperationCosts 获取所有当前活跃的操作成本记录（需要 owner/admin）。
func (a *api) getAdminOperationCosts(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	if a.walletService == nil || a.walletService.GetOperationCostRepository() == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "wallet service not configured")
		return
	}

	costs, err := a.walletService.GetOperationCostRepository().GetAllCosts(r.Context(), auth.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	out := make([]operationCostAdminResponse, 0, len(costs))
	for _, cost := range costs {
		if cost != nil {
			out = append(out, operationCostAdminDTO(cost))
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"operation_costs": out})
}

// updateOperationCostsRequest 批量更新请求体。
type updateOperationCostsRequest struct {
	Updates []updateOperationCostRequest `json:"updates"`
	Reason  string                       `json:"reason,omitempty"`
}

// UpdateAdminOperationCosts 批量更新操作成本（需要 owner/admin）。
func (a *api) updateAdminOperationCosts(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	var req updateOperationCostsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if len(req.Updates) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "updates array is empty")
		return
	}

	if a.walletService == nil || a.walletService.GetOperationCostRepository() == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "wallet service not configured")
		return
	}

	ocr := a.walletService.GetOperationCostRepository()

	// 获取当前成本记录
	currentCosts := make(map[string]*domain.OperationCostRow)
	for _, update := range req.Updates {
		opType := domain.OperationType(update.OperationType)
		if !isValidOperationType(opType) {
			writeError(w, http.StatusBadRequest, "invalid_operation_type", "unknown operation type: "+update.OperationType)
			return
		}

		existing, err := ocr.GetCost(r.Context(), auth.OrganizationID, opType)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		currentCosts[update.OperationType] = existing
	}

	// 逐个更新
	updatedCosts := make([]operationCostAdminResponse, 0, len(req.Updates))
	for _, update := range req.Updates {
		opType := domain.OperationType(update.OperationType)
		existing := currentCosts[update.OperationType]

		// 创建新记录
		newCost := &domain.OperationCostRow{
			OperationType:  opType,
			OrganizationID: auth.OrganizationID,
			CreditsCost:    update.CreditsCost,
			EffectiveAt:    time.Now().Unix(),
			CreatedAt:      time.Now().Unix(),
			UpdatedAt:      time.Now().Unix(),
		}

		// 更新（创建新记录并写入历史）
		if err := ocr.UpdateCost(r.Context(), existing, newCost, req.Reason, auth.UserID); err != nil {
			writeServiceError(w, err)
			return
		}

		updatedCosts = append(updatedCosts, operationCostAdminDTO(newCost))
	}

	writeJSON(w, http.StatusOK, map[string]any{"operation_costs": updatedCosts})
}

// getAdminOperationCostHistory 获取指定操作类型的修改历史（需要 owner/admin）。
func (a *api) getAdminOperationCostHistory(w http.ResponseWriter, r *http.Request) {
	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	opTypeStr := chi.URLParam(r, "operationType")
	opType := domain.OperationType(opTypeStr)

	if !isValidOperationType(opType) {
		writeError(w, http.StatusBadRequest, "invalid_operation_type", "unknown operation type: "+opTypeStr)
		return
	}

	if a.walletService == nil || a.walletService.GetOperationCostRepository() == nil {
		writeError(w, http.StatusInternalServerError, "service_not_available", "wallet service not configured")
		return
	}

	// 获取历史记录（暂时从仓库获取）
	history, err := a.walletService.GetOperationCostRepository().GetCostHistory(r.Context(), auth.OrganizationID, opType)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	out := make([]operationCostHistoryResponse, 0, len(history))
	for _, h := range history {
		if h != nil {
			out = append(out, operationCostHistoryDTO(h))
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"history": out})
}

// isValidOperationType 检查操作类型是否有效。
func isValidOperationType(opType domain.OperationType) bool {
	switch opType {
	case domain.OperationTypeStoryAnalysis,
		domain.OperationTypeImageGeneration,
		domain.OperationTypeVideoGeneration,
		domain.OperationTypeChat,
		domain.OperationTypeStoryboardEdit,
		domain.OperationTypeCharacterEdit,
		domain.OperationTypeSceneEdit:
		return true
	}
	return false
}

// extractRequestAuthFromContext 从请求中提取认证信息，并在失败时写入错误响应。
func extractRequestAuthFromContext(w http.ResponseWriter, r *http.Request) (service.RequestAuthContext, bool) {
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return service.RequestAuthContext{}, false
	}
	if auth.OrganizationID == "" {
		writeError(w, http.StatusForbidden, "forbidden", "organization context required")
		return service.RequestAuthContext{}, false
	}
	return auth, true
}
