package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

// initiateChargeRequest 支付初始化请求
type initiateChargeRequest struct {
	Amount   int64  `json:"amount"`   // 金额（分）
	Currency string `json:"currency"` // 货币
}

// initiateChargeResponse 支付初始化响应
type initiateChargeResponse struct {
	SessionID string `json:"sessionId"` // 支付会话 ID
	URL       string `json:"url"`       // 支付网关重定向 URL
	OrderID   string `json:"orderId"`   // 订单 ID
}

// initiateChargeWallet POST /api/v1/wallet:charge:initiate
// 初始化支付，返回支付网关 URL 用于重定向
func (a *api) initiateChargeWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 从上下文获取用户信息
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated")
		return
	}

	// 解析请求
	var req initiateChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// 验证请求参数
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_amount", "amount must be greater than 0")
		return
	}
	if req.Currency == "" {
		writeError(w, http.StatusBadRequest, "invalid_currency", "currency is required")
		return
	}

	// 生成订单 ID
	orderID, err := domain.NewID()
	if err != nil {
		a.logger.Error("failed to generate order ID", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate order ID")
		return
	}
	orderID = "order_" + orderID

	// 调用支付服务初始化支付
	session, err := a.paymentService.InitiateCharge(ctx, auth.UserID, auth.OrganizationID, orderID, req.Amount, req.Currency)
	if err != nil {
		a.logger.Error("failed to initiate charge", slog.String("error", err.Error()))
		writeServiceError(w, err)
		return
	}

	// 返回支付信息
	resp := initiateChargeResponse{
		SessionID: session.SessionID,
		URL:       session.URL,
		OrderID:   session.OrderID,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePaymentWebhook POST /webhook/payment
// 处理支付网关回调（Stripe webhook）
func (a *api) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 验证 webhook 签名并解析 payload
	payload, err := a.paymentService.Provider().VerifyWebhook(r)
	if err != nil {
		a.logger.Error("failed to verify webhook signature", slog.String("error", err.Error()))
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid webhook signature")
		return
	}

	// 处理 webhook
	if err := a.paymentService.ProcessWebhook(ctx, payload); err != nil {
		a.logger.Error("failed to process payment webhook", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "processing_error", "failed to process payment")
		return
	}

	// 返回成功（200 OK，告诉 Stripe 收到了）
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
