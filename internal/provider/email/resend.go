package email

import (
	"context"
	"fmt"

	"github.com/resendlabs/resend-go"
)

// ResendEmailProvider Resend 邮件提供商
type ResendEmailProvider struct {
	client *resend.Client
	from   string
}

// NewResendEmailProvider 创建 Resend 邮件提供商
func NewResendEmailProvider(apiKey, fromEmail string) *ResendEmailProvider {
	return &ResendEmailProvider{
		client: resend.NewClient(apiKey),
		from:   fromEmail,
	}
}

// EmailMessage 邮件消息
type EmailMessage struct {
	To       string // 收件人邮箱
	Subject  string
	HTML     string
	TextBody string // 可选，纯文本版本
}

// Send 发送单个邮件
func (p *ResendEmailProvider) Send(ctx context.Context, msg EmailMessage) (messageID string, err error) {
	params := &resend.SendEmailRequest{
		From:    p.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
	}

	if msg.TextBody != "" {
		params.Text = msg.TextBody
	}

	resp, err := p.client.Emails.Send(params)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	if resp.Id == "" {
		return "", fmt.Errorf("no message id from Resend API")
	}

	return resp.Id, nil
}

// SendBatch 批量发送邮件（同步）
func (p *ResendEmailProvider) SendBatch(ctx context.Context, messages []EmailMessage) ([]string, error) {
	var messageIDs []string
	var lastErr error

	for _, msg := range messages {
		id, err := p.Send(ctx, msg)
		if err != nil {
			lastErr = err
			// 继续发送其他邮件
			messageIDs = append(messageIDs, "")
			continue
		}
		messageIDs = append(messageIDs, id)
	}

	return messageIDs, lastErr
}
