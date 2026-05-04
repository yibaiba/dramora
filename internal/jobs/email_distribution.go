package jobs

import (
	"context"
	"time"
)

// EmailDistributionPayload 邮件分发任务负载
type EmailDistributionPayload struct {
	// 基本信息
	OrganizationID string   `json:"organization_id"`
	Codes          []string `json:"codes"`      // 赎回码列表
	Recipients     []string `json:"recipients"` // 收件人邮箱列表

	// 邮件内容
	Subject  string `json:"subject"`   // 邮件主题（支持模板）
	HTMLBody string `json:"html_body"` // 自定义 HTML（如果为空使用默认模板）

	// 可选参数
	CampaignName string `json:"campaign_name,omitempty"`
	CampaignID   string `json:"campaign_id,omitempty"`
	SenderName   string `json:"sender_name,omitempty"`

	// 创建时间（用于失败重试）
	CreatedAt time.Time `json:"created_at"`
	Attempts  int       `json:"attempts"`
}

// EmailDistributionExecutor 邮件分发任务执行器
type EmailDistributionExecutor interface {
	// ProcessEmailDistribution 处理邮件分发任务
	ProcessEmailDistribution(ctx context.Context, payload EmailDistributionPayload) error
}
