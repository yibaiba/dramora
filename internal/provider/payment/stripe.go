package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/webhook"
)

// StripeProvider 实现 Stripe 支付网关适配
type StripeProvider struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
}

// NewStripeProvider 创建 Stripe provider
func NewStripeProvider(secretKey, webhookSecret, successURL, cancelURL string) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

// CreateSession 创建 Stripe checkout 会话
func (p *StripeProvider) CreateSession(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error) {
	// 转换为 Stripe 单位（美元为例）
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String("price_free"),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(p.successURL + "?order_id=" + req.OrderID),
		CancelURL:  stripe.String(p.cancelURL + "?order_id=" + req.OrderID),
	}

	sess, err := session.New(params)
	if err != nil {
		return CreateSessionResponse{}, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return CreateSessionResponse{
		URL:       sess.URL,
		SessionID: sess.ID,
	}, nil
}

// VerifyWebhook 验证并解析 Stripe webhook
func (p *StripeProvider) VerifyWebhook(r *http.Request) (WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return WebhookPayload{}, fmt.Errorf("failed to read webhook body: %w", err)
	}

	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, signatureHeader, p.webhookSecret)
	if err != nil {
		return WebhookPayload{}, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	// 仅处理 checkout.session.completed 和 charge.failed 事件
	if event.Type == "checkout.session.completed" {
		var checkoutSession struct {
			ID          string            `json:"id"`
			AmountTotal int64             `json:"amount_total"`
			Created     int64             `json:"created"`
			Metadata    map[string]string `json:"metadata"`
		}
		err = json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			return WebhookPayload{}, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		orderID := ""
		if checkoutSession.Metadata != nil {
			orderID = checkoutSession.Metadata["order_id"]
		}
		return WebhookPayload{
			OrderID:   orderID,
			SessionID: checkoutSession.ID,
			Status:    "success",
			Amount:    checkoutSession.AmountTotal,
			Timestamp: checkoutSession.Created,
		}, nil
	}

	if event.Type == "charge.failed" {
		var charge struct {
			ID              string            `json:"id"`
			Amount          int64             `json:"amount"`
			Created         int64             `json:"created"`
			PaymentIntentID string            `json:"payment_intent"`
			Metadata        map[string]string `json:"metadata"`
		}
		err = json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			return WebhookPayload{}, fmt.Errorf("failed to parse charge: %w", err)
		}

		orderID := ""
		if charge.Metadata != nil {
			orderID = charge.Metadata["order_id"]
		}
		return WebhookPayload{
			OrderID:   orderID,
			SessionID: charge.PaymentIntentID,
			Status:    "failed",
			Amount:    charge.Amount,
			Timestamp: charge.Created,
		}, nil
	}

	// 其他事件类型返回空状态
	return WebhookPayload{
		Status: "unknown",
	}, nil
}
