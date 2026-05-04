package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

// redemptionCampaignDTO 赎回活动 DTO
type redemptionCampaignDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      string `json:"created_at"`
	OrganizationID string `json:"organization_id"`
}

// redemptionCodeDTO 赎回码 DTO
type redemptionCodeDTO struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	OrganizationID string `json:"organization_id"`
	CampaignID     string `json:"campaign_id,omitempty"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      string `json:"created_at"`
	UsedBy         string `json:"used_by,omitempty"`
	UsedAt         string `json:"used_at,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// campaignStatsDTO 活动统计 DTO
type campaignStatsDTO struct {
	CampaignID      string  `json:"campaign_id"`
	Name            string  `json:"name"`
	TotalCodes      int64   `json:"total_codes"`
	UsedCodes       int64   `json:"used_codes"`
	UnusedCodes     int64   `json:"unused_codes"`
	TotalAmount     int64   `json:"total_amount"`
	UsedAmount      int64   `json:"used_amount"`
	UnusedAmount    int64   `json:"unused_amount"`
	UsagePercentage float64 `json:"usage_percentage"`
}

// generateCodesRequest 生成赎回码的请求
type generateCodesRequest struct {
	Count      int        `json:"count"`
	Amount     int64      `json:"amount"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	CampaignID string     `json:"campaign_id,omitempty"`
}

// redeemCodeRequest 兑换赎回码的请求
type redeemCodeRequest struct {
	Code string `json:"code"`
}

// createCampaignRequest 创建赎回活动的请求
type createCampaignRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func campaignToDTO(c *domain.RedemptionCampaign) redemptionCampaignDTO {
	return redemptionCampaignDTO{
		ID:             c.ID,
		Name:           c.Name,
		Description:    c.Description,
		Status:         string(c.Status),
		CreatedBy:      c.CreatedBy,
		CreatedAt:      c.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		OrganizationID: c.OrganizationID,
	}
}

func codeToDTO(rc *domain.RedemptionCode) redemptionCodeDTO {
	dto := redemptionCodeDTO{
		ID:             rc.ID,
		Code:           rc.Code,
		OrganizationID: rc.OrganizationID,
		CampaignID:     rc.CampaignID,
		Amount:         rc.Amount,
		Status:         string(rc.Status),
		CreatedBy:      rc.CreatedBy,
		CreatedAt:      rc.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		UsedBy:         rc.UsedBy,
		Reason:         rc.Reason,
	}
	if !rc.UsedAt.IsZero() {
		dto.UsedAt = rc.UsedAt.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if !rc.ExpiresAt.IsZero() {
		dto.ExpiresAt = rc.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return dto
}

func statsToDTO(s *domain.CampaignStats) campaignStatsDTO {
	return campaignStatsDTO{
		CampaignID:      s.CampaignID,
		Name:            s.Name,
		TotalCodes:      s.TotalCodes,
		UsedCodes:       s.UsedCodes,
		UnusedCodes:     s.UnusedCodes,
		TotalAmount:     s.TotalAmount,
		UsedAmount:      s.UsedAmount,
		UnusedAmount:    s.UnusedAmount,
		UsagePercentage: s.UsagePercentage,
	}
}

// generateCodes 生成赎回码 (Admin)
// POST /api/v1/admin/redemption-codes:generate
func (a *api) generateCodes(w http.ResponseWriter, r *http.Request) {
	if a.redemptionCodeService == nil {
		writeError(w, http.StatusServiceUnavailable, "redemption_unavailable", "redemption code service not configured")
		return
	}

	var req generateCodesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	request := service.GenerateCodeRequest{
		Count:      req.Count,
		Amount:     req.Amount,
		Reason:     req.Reason,
		CampaignID: req.CampaignID,
	}
	if req.ExpiresAt != nil {
		request.ExpiresAt = *req.ExpiresAt
	}

	codes, err := a.redemptionCodeService.GenerateCodes(r.Context(), auth, request)
	if err != nil {
		writeRedemptionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{
		"codes": codes,
		"count": len(codes),
	})
}

// createCampaign 创建赎回活动 (Admin)
// POST /api/v1/admin/redemption-campaigns:create
func (a *api) createCampaign(w http.ResponseWriter, r *http.Request) {
	if a.redemptionCodeService == nil {
		writeError(w, http.StatusServiceUnavailable, "redemption_unavailable", "redemption code service not configured")
		return
	}

	var req createCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	campaign, err := a.redemptionCodeService.CreateCampaign(r.Context(), auth, service.CreateCampaignRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeRedemptionError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, Envelope{
		"campaign": campaignToDTO(campaign),
	})
}

// redeemCode 用户兑换赎回码
// POST /api/v1/redemption-codes:redeem
func (a *api) redeemCode(w http.ResponseWriter, r *http.Request) {
	if a.redemptionCodeService == nil {
		writeError(w, http.StatusServiceUnavailable, "redemption_unavailable", "redemption code service not configured")
		return
	}

	var req redeemCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	resp, err := a.redemptionCodeService.RedeemCode(r.Context(), auth, req.Code)
	if err != nil {
		writeRedemptionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// getCampaignStats 获取活动统计数据 (Admin)
// GET /api/v1/admin/redemption-campaigns/{campaignId}/stats
func (a *api) getCampaignStats(w http.ResponseWriter, r *http.Request) {
	if a.redemptionCodeService == nil {
		writeError(w, http.StatusServiceUnavailable, "redemption_unavailable", "redemption code service not configured")
		return
	}

	campaignID := chi.URLParam(r, "campaignId")
	if campaignID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "campaign_id is required")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	stats, err := a.redemptionCodeService.GetCampaignStats(r.Context(), auth, campaignID)
	if err != nil {
		writeRedemptionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{
		"stats": statsToDTO(stats),
	})
}

// writeRedemptionError 将赎回码错误映射到 HTTP 响应
func writeRedemptionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCodeNotFound):
		writeError(w, http.StatusNotFound, "code_not_found", "赎回码不存在")
	case errors.Is(err, domain.ErrCodeAlreadyUsed):
		writeError(w, http.StatusUnprocessableEntity, "code_already_used", "赎回码已被使用")
	case errors.Is(err, domain.ErrCodeExpired):
		writeError(w, http.StatusUnprocessableEntity, "code_expired", "赎回码已过期")
	case errors.Is(err, domain.ErrCampaignNotFound):
		writeError(w, http.StatusNotFound, "campaign_not_found", "赎回活动不存在")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusForbidden, "unauthorized", "权限不足")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "内部服务错误")
	}
}
