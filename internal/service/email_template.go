package service

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

// EmailParams 邮件模板参数
type EmailParams struct {
	Code          string    // 赎回码
	ShortURL      string    // 短链接
	ExpiresAt     time.Time // 过期时间
	Amount        int64     // 积分金额
	CampaignName  string    // 活动名称
	RecipientName string    // 收件人名称
	QRCodeBase64  string    // QR码 Base64 编码（可选）
}

// TemplateData 完整的模板数据
type TemplateData struct {
	Code          string
	ShortURL      string
	ExpiresAt     string
	ExpiresAtDate string // 仅日期格式
	Amount        string
	CampaignName  string
	RecipientName string
	QRCodeDataURL string // Data URL 格式
}

// DefaultHTMLTemplate 默认的 HTML 邮件模板
const DefaultHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .code-section { background: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0; text-align: center; }
        .code-value { font-size: 24px; font-weight: bold; font-family: monospace; letter-spacing: 2px; }
        .link-section { margin: 20px 0; }
        .qr-section { text-align: center; margin: 30px 0; }
        .qr-section img { max-width: 200px; }
        .expiry { color: #999; font-size: 12px; margin-top: 20px; }
        .footer { text-align: center; margin-top: 40px; border-top: 1px solid #ddd; padding-top: 20px; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎁 您的赎回码已准备好</h1>
        </div>

        <p>Hi {{ .RecipientName }}!</p>

        <p>感谢您！我们为您准备了{{ .Amount }}积分，可通过以下任一方式兑换：</p>

        <!-- 方式 1: 直接输入赎回码 -->
        <div class="code-section">
            <p style="margin: 0 0 10px 0; color: #999;">赎回码</p>
            <div class="code-value">{{ .Code }}</div>
            <p style="margin: 10px 0 0 0; color: #999; font-size: 12px;">复制此码在兑换页面输入</p>
        </div>

        <!-- 方式 2: 点击链接 -->
        <div class="link-section">
            <p>或者直接点击此链接兑换:</p>
            <p><a href="{{ .ShortURL }}" style="background: #007bff; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; display: inline-block;">立即兑换</a></p>
        </div>

        <!-- 方式 3: 扫码 -->
        {{- if .QRCodeDataURL }}
        <div class="qr-section">
            <p>或扫码兑换：</p>
            <img src="{{ .QRCodeDataURL }}" alt="QR Code" />
        </div>
        {{- end }}

        <p class="expiry">⏰ 有效期至: {{ .ExpiresAtDate }}</p>

        {{- if .CampaignName }}
        <p style="font-size: 12px; color: #999;">活动: {{ .CampaignName }}</p>
        {{- end }}

        <div class="footer">
            <p>此链接仅对您有效。如有任何问题，请联系我们的支持团队。</p>
        </div>
    </div>
</body>
</html>`

// EmailTemplateService 邮件模板服务
type EmailTemplateService struct {
	htmlTemplate *template.Template
}

// NewEmailTemplateService 创建邮件模板服务
func NewEmailTemplateService() (*EmailTemplateService, error) {
	tmpl, err := template.New("email").Parse(DefaultHTMLTemplate)
	if err != nil {
		return nil, err
	}

	return &EmailTemplateService{
		htmlTemplate: tmpl,
	}, nil
}

// RenderHTML 渲染 HTML 邮件内容
func (s *EmailTemplateService) RenderHTML(params EmailParams) (string, error) {
	// 转换参数为模板数据
	data := TemplateData{
		Code:          params.Code,
		ShortURL:      params.ShortURL,
		Amount:        fmt.Sprintf("%d", params.Amount),
		RecipientName: params.RecipientName,
		CampaignName:  params.CampaignName,
	}

	// 格式化过期时间
	if !params.ExpiresAt.IsZero() {
		data.ExpiresAt = params.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000Z")
		data.ExpiresAtDate = params.ExpiresAt.Format("2006-01-02")
	}

	// 如果有 QR 码，转换为 Data URL
	if params.QRCodeBase64 != "" {
		data.QRCodeDataURL = "data:image/png;base64," + params.QRCodeBase64
	}

	// 渲染模板
	var buf strings.Builder
	if err := s.htmlTemplate.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderSubject 渲染邮件主题（支持简单的模板替换）
func (s *EmailTemplateService) RenderSubject(subjectTpl string, params EmailParams) (string, error) {
	tmpl, err := template.New("subject").Parse(subjectTpl)
	if err != nil {
		return "", err
	}

	data := TemplateData{
		Code:          params.Code,
		ShortURL:      params.ShortURL,
		Amount:        fmt.Sprintf("%d", params.Amount),
		RecipientName: params.RecipientName,
		CampaignName:  params.CampaignName,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
