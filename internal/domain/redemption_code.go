package domain

import (
	"errors"
	"time"
)

// RedemptionCodeStatus 定义赎回码状态常量
type RedemptionCodeStatus string

const (
	// RedemptionCodeStatusUnused 未使用的赎回码
	RedemptionCodeStatusUnused RedemptionCodeStatus = "unused"
	// RedemptionCodeStatusUsed 已使用的赎回码
	RedemptionCodeStatusUsed RedemptionCodeStatus = "used"
)

// RedemptionCampaignStatus 定义赎回活动状态常量
type RedemptionCampaignStatus string

const (
	// RedemptionCampaignStatusActive 活跃的活动
	RedemptionCampaignStatusActive RedemptionCampaignStatus = "active"
	// RedemptionCampaignStatusInactive 非活跃的活动
	RedemptionCampaignStatusInactive RedemptionCampaignStatus = "inactive"
	// RedemptionCampaignStatusExpired 已过期的活动
	RedemptionCampaignStatusExpired RedemptionCampaignStatus = "expired"
)

// RedemptionCode 代表一个赎回码记录
type RedemptionCode struct {
	ID             string
	Code           string // 赎回码，如 GIFT-XXXXX
	OrganizationID string
	CampaignID     string // 关联的活动 ID（可选）
	Amount         int64  // 积分金额
	Status         RedemptionCodeStatus
	CreatedBy      string // 创建者 user_id
	CreatedAt      time.Time
	UsedBy         string // 兑换者 user_id
	UsedAt         time.Time
	ExpiresAt      time.Time // 过期时间（零值 = 永不过期）
	Reason         string    // 备注
	Version        int       // 乐观锁版本号
}

// RedemptionCampaign 代表一个赎回活动
type RedemptionCampaign struct {
	ID             string
	Name           string
	Description    string
	Status         RedemptionCampaignStatus
	CreatedBy      string
	CreatedAt      time.Time
	OrganizationID string
}

// CampaignStats 代表活动的统计数据
type CampaignStats struct {
	CampaignID      string
	Name            string
	TotalCodes      int64
	UsedCodes       int64
	UnusedCodes     int64
	TotalAmount     int64
	UsedAmount      int64
	UnusedAmount    int64
	UsagePercentage float64
}

// Validate 检查赎回码的合法性
func (rc *RedemptionCode) Validate() error {
	if rc.Code == "" {
		return ErrInvalidInput
	}
	if rc.OrganizationID == "" {
		return ErrInvalidInput
	}
	if rc.Amount <= 0 {
		return ErrInvalidInput
	}
	if rc.CreatedBy == "" {
		return ErrInvalidInput
	}
	return nil
}

// Validate 检查赎回活动的合法性
func (c *RedemptionCampaign) Validate() error {
	if c.Name == "" {
		return ErrInvalidInput
	}
	if c.OrganizationID == "" {
		return ErrInvalidInput
	}
	if c.CreatedBy == "" {
		return ErrInvalidInput
	}
	return nil
}

// IsExpired 检查赎回码是否已过期
func (rc *RedemptionCode) IsExpired() bool {
	if rc.ExpiresAt.IsZero() {
		return false // 零值表示永不过期
	}
	return time.Now().UTC().After(rc.ExpiresAt)
}

// IsValid 检查赎回码是否有效（未使用且未过期）
func (rc *RedemptionCode) IsValid() bool {
	if rc.Status != RedemptionCodeStatusUnused {
		return false
	}
	return !rc.IsExpired()
}

// ShortCode 代表一个短链接记录
type ShortCode struct {
	ID             int64
	Code           string // 赎回码，如 GIFT-XXXXX
	ShortCode      string // Base62 编码的短码
	OrganizationID string
	CreatedAt      time.Time
}

// Redemption-specific errors
var (
	// ErrCodeNotFound 赎回码不存在
	ErrCodeNotFound = errors.New("redemption code not found")
	// ErrCodeAlreadyUsed 赎回码已被使用
	ErrCodeAlreadyUsed = errors.New("redemption code already used")
	// ErrCodeExpired 赎回码已过期
	ErrCodeExpired = errors.New("redemption code expired")
	// ErrCodeVersionConflict 版本号冲突（乐观锁）
	ErrCodeVersionConflict = errors.New("redemption code version conflict")
	// ErrCampaignNotFound 赎回活动不存在
	ErrCampaignNotFound = errors.New("redemption campaign not found")
	// ErrShortCodeNotFound 短链接不存在
	ErrShortCodeNotFound = errors.New("short code not found")
)
