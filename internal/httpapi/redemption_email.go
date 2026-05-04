package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/pkg/base62"
	"github.com/yibaiba/dramora/internal/pkg/qrcode"
)

// sendEmailRequest 邮件分发请求
type sendEmailRequest struct {
	Codes        []string `json:"codes"`               // 赎回码列表
	Recipients   []string `json:"recipients"`          // 收件人邮箱列表
	Subject      string   `json:"subject"`             // 邮件主题
	HTMLBody     string   `json:"html_body,omitempty"` // 自定义 HTML（可选）
	CampaignName string   `json:"campaign_name,omitempty"`
	CampaignID   string   `json:"campaign_id,omitempty"`
}

// sendEmailResponse 邮件分发响应
type sendEmailResponse struct {
	TaskID string `json:"task_id"` // 后台任务 ID
}

// shortCodeResponse 短链接响应
type shortCodeResponse struct {
	Code      string `json:"code"`
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	QRCode    string `json:"qr_code"` // Base64 编码的 QR 码 PNG
}

// sendEmail 邮件分发 (Admin)
// POST /api/v1/admin/redemption-codes:send-email
func (a *api) sendEmail(w http.ResponseWriter, r *http.Request) {
	if a.shortCodeRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "email_unavailable", "email service not configured")
		return
	}

	var req sendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// 验证请求参数
	if len(req.Codes) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "codes is required")
		return
	}
	if len(req.Recipients) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "recipients is required")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "subject is required")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	// 为每个赎回码生成短链接
	for _, code := range req.Codes {
		// 检查是否已存在短链接
		existing, err := a.shortCodeRepo.GetByCode(r.Context(), code)
		if err == nil && existing != nil {
			// 短链接已存在，跳过
			continue
		}

		// 生成新的短链接
		// 这里使用 ID 作为基础，实际会在创建时由数据库生成
		// 暂时占位符：后续会在数据库层更新为实际的 ID 计算
		shortCode := base62.Encode(uint64(len(req.Codes)))

		// 创建短链接记录
		sc := &domain.ShortCode{
			Code:           code,
			ShortCode:      shortCode,
			OrganizationID: auth.OrganizationID,
		}

		if err := a.shortCodeRepo.Create(r.Context(), sc); err != nil {
			// 记录错误但继续
			continue
		}
	}

	// 构建后台任务
	payload := jobs.EmailDistributionPayload{
		OrganizationID: auth.OrganizationID,
		Codes:          req.Codes,
		Recipients:     req.Recipients,
		Subject:        req.Subject,
		HTMLBody:       req.HTMLBody,
		CampaignName:   req.CampaignName,
		CampaignID:     req.CampaignID,
		CreatedAt:      time.Now(),
		Attempts:       0,
	}

	// 投入任务队列
	job := jobs.Job{
		ID:      generateID(), // 生成任务 ID
		Kind:    jobs.JobKindEmailDistribution,
		Payload: toMap(payload),
	}

	if err := a.jobsClient.Enqueue(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to queue email job")
		return
	}

	writeJSON(w, http.StatusAccepted, sendEmailResponse{
		TaskID: job.ID,
	})
}

// shortenCode 生成短链接和 QR 码
// POST /api/v1/redemption-codes:shorten
func (a *api) shortenCode(w http.ResponseWriter, r *http.Request) {
	if a.shortCodeRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "shortcode_unavailable", "short code service not configured")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "code is required")
		return
	}

	auth, ok := extractRequestAuthFromContext(w, r)
	if !ok {
		return
	}

	// 检查是否已存在短链接
	existing, err := a.shortCodeRepo.GetByCode(r.Context(), req.Code)
	if err == nil && existing != nil {
		// 短链接已存在，返回现有的
		shortURL := "https://dramora.app/r/" + existing.ShortCode
		qrBase64, _ := qrcode.GenerateQRCodeBase64(shortURL)

		writeJSON(w, http.StatusOK, shortCodeResponse{
			Code:      existing.Code,
			ShortCode: existing.ShortCode,
			ShortURL:  shortURL,
			QRCode:    qrBase64,
		})
		return
	}

	// 生成新的短链接
	// 使用简单的 counter 作为 ID 基础
	shortCode := base62.Encode(uint64(len(req.Code)))

	sc := &domain.ShortCode{
		Code:           req.Code,
		ShortCode:      shortCode,
		OrganizationID: auth.OrganizationID,
	}

	if err := a.shortCodeRepo.Create(r.Context(), sc); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create short code")
		return
	}

	// 生成 QR 码
	shortURL := "https://dramora.app/r/" + sc.ShortCode
	qrBase64, err := qrcode.GenerateQRCodeBase64(shortURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate QR code")
		return
	}

	writeJSON(w, http.StatusCreated, shortCodeResponse{
		Code:      sc.Code,
		ShortCode: sc.ShortCode,
		ShortURL:  shortURL,
		QRCode:    qrBase64,
	})
}

// redirectShortCode 短链接重定向
// GET /r/:shortCode
func (a *api) redirectShortCode(w http.ResponseWriter, r *http.Request) {
	if a.shortCodeRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "shortcode_unavailable", "short code service not configured")
		return
	}

	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "short code is required")
		return
	}

	// 查询短链接
	sc, err := a.shortCodeRepo.GetByShortCode(r.Context(), shortCode)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "short code not found")
		return
	}

	// 重定向到兑换页面
	redirectURL := "https://dramora.app/redeem?code=" + sc.Code
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

// toMap 将结构体转换为 map[string]any
func toMap(v interface{}) map[string]any {
	// 使用 JSON 作为中介进行序列化/反序列化
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

// generateID 生成唯一的 ID
func generateID() string {
	id, err := domain.NewID()
	if err != nil {
		// 生成失败时返回空字符串（这不应该发生）
		return ""
	}
	return id
}
