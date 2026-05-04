package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// RedemptionCodeService 提供赎回码相关业务逻辑
type RedemptionCodeService struct {
	repo       repo.RedemptionCodeRepository
	walletSvc  *WalletService
	walletRepo repo.WalletRepository
}

// NewRedemptionCodeService 创建 RedemptionCodeService 实例
func NewRedemptionCodeService(
	r repo.RedemptionCodeRepository,
	w *WalletService,
	wr repo.WalletRepository,
) *RedemptionCodeService {
	return &RedemptionCodeService{
		repo:       r,
		walletSvc:  w,
		walletRepo: wr,
	}
}

// GenerateCodeRequest 生成赎回码的请求参数
type GenerateCodeRequest struct {
	Count      int       // 生成数量
	Amount     int64     // 每个码的积分金额
	ExpiresAt  time.Time // 过期时间（零值 = 永不过期）
	Reason     string    // 备注说明
	CampaignID string    // 关联的活动 ID（可选）
}

// Validate 检查请求参数合法性
func (req GenerateCodeRequest) Validate() error {
	if req.Count <= 0 || req.Count > 10000 {
		return errors.New("count must be between 1 and 10000")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	return nil
}

// GenerateCodes 生成批量赎回码
func (s *RedemptionCodeService) GenerateCodes(ctx context.Context, authCtx RequestAuthContext, req GenerateCodeRequest) ([]string, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 检查权限（需要 admin/owner 角色）
	if authCtx.Role != "owner" && authCtx.Role != "admin" {
		return nil, ErrUnauthorized
	}

	// 如果提供了 campaignID，验证活动是否存在
	if req.CampaignID != "" {
		campaign, err := s.repo.GetCampaignByID(ctx, req.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("invalid campaign: %w", err)
		}
		if campaign.OrganizationID != authCtx.OrganizationID {
			return nil, errors.New("campaign not in this organization")
		}
	}

	// 生成赎回码
	codes := make([]*domain.RedemptionCode, req.Count)
	codeStrings := make([]string, req.Count)

	for i := 0; i < req.Count; i++ {
		codeStr := generateRandomCode()
		codeID, err := domain.NewID()
		if err != nil {
			return nil, fmt.Errorf("generate code id: %w", err)
		}

		codes[i] = &domain.RedemptionCode{
			ID:             codeID,
			Code:           codeStr,
			OrganizationID: authCtx.OrganizationID,
			CampaignID:     req.CampaignID,
			Amount:         req.Amount,
			Status:         domain.RedemptionCodeStatusUnused,
			CreatedBy:      authCtx.UserID,
			CreatedAt:      time.Now().UTC(),
			ExpiresAt:      req.ExpiresAt,
			Reason:         req.Reason,
			Version:        0,
		}

		codeStrings[i] = codeStr
	}

	// 批量插入到数据库
	if err := s.repo.GenerateCodes(ctx, codes); err != nil {
		return nil, fmt.Errorf("generate codes in db: %w", err)
	}

	return codeStrings, nil
}

// RedeemCodeRequest 兑换赎回码的请求参数
type RedeemCodeRequest struct {
	Code string // 赎回码
}

// RedeemCodeResponse 兑换赎回码的响应
type RedeemCodeResponse struct {
	Success    bool   `json:"success"`
	Amount     int64  `json:"amount"`
	NewBalance int64  `json:"new_balance"`
	Message    string `json:"message"`
}

// validateAndGetCode 验证并获取赎回码
func (s *RedemptionCodeService) validateAndGetCode(ctx context.Context, authCtx RequestAuthContext, codeStr string) (*domain.RedemptionCode, error) {
	if codeStr = strings.TrimSpace(codeStr); codeStr == "" {
		return nil, errors.New("code cannot be empty")
	}

	// 获取赎回码
	code, err := s.repo.GetCodeByCode(ctx, codeStr)
	if err != nil {
		if errors.Is(err, domain.ErrCodeNotFound) {
			return nil, domain.ErrCodeNotFound
		}
		return nil, err
	}

	// 检查组织归属
	if code.OrganizationID != authCtx.OrganizationID {
		return nil, domain.ErrCodeNotFound // 不泄露组织间码信息
	}

	// 检查状态和过期时间
	if code.Status != domain.RedemptionCodeStatusUnused {
		return nil, domain.ErrCodeAlreadyUsed
	}

	if code.IsExpired() {
		return nil, domain.ErrCodeExpired
	}

	return code, nil
}

// attemptRedeemWithRetry 执行兑换并带重试机制
func (s *RedemptionCodeService) attemptRedeemWithRetry(ctx context.Context, authCtx RequestAuthContext, code *domain.RedemptionCode, codeStr string) (*domain.RedemptionCode, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		redeemedCode, err := s.repo.RedeemCode(ctx, codeStr, authCtx.UserID, code.Version)
		if err == nil {
			return redeemedCode, nil
		}

		// 处理不同的错误
		if errors.Is(err, domain.ErrCodeAlreadyUsed) {
			return nil, domain.ErrCodeAlreadyUsed
		}
		if errors.Is(err, domain.ErrCodeExpired) {
			return nil, domain.ErrCodeExpired
		}
		if errors.Is(err, domain.ErrCodeNotFound) {
			return nil, domain.ErrCodeNotFound
		}

		// 版本冲突，重新查询后重试
		if errors.Is(err, domain.ErrCodeVersionConflict) {
			lastErr = err
			// 重新查询最新的码信息
			updatedCode, err := s.repo.GetCodeByCode(ctx, codeStr)
			if err != nil {
				return nil, err
			}

			// 如果已被使用，不再重试
			if updatedCode.Status != domain.RedemptionCodeStatusUnused {
				return nil, domain.ErrCodeAlreadyUsed
			}

			code = updatedCode
			continue
		}

		return nil, err
	}

	return nil, fmt.Errorf("failed to redeem code after %d attempts: %w", maxRetries, lastErr)
}

// recordRedemptionTransaction 记录兑换交易到钱包
func (s *RedemptionCodeService) recordRedemptionTransaction(ctx context.Context, authCtx RequestAuthContext, redeemedCode *domain.RedemptionCode, codeStr string) (domain.Wallet, error) {
	reason := fmt.Sprintf("redemption code: %s", codeStr[len(codeStr)-4:])

	wallet, _, err := s.walletRepo.ApplyTransaction(ctx, repo.WalletApplyParams{
		OrganizationID: authCtx.OrganizationID,
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         redeemedCode.Amount,
		Reason:         reason,
		RefType:        "redemption_code",
		RefID:          redeemedCode.ID,
		ActorUserID:    authCtx.UserID,
		TransactionID:  "",
		CreatedAt:      time.Now().UTC(),
	})

	if err != nil {
		return domain.Wallet{}, fmt.Errorf("record wallet transaction: %w", err)
	}

	return wallet, nil
}

// RedeemCode 用户兑换赎回码
// 使用乐观锁 + 重试机制确保幂等性
func (s *RedemptionCodeService) RedeemCode(ctx context.Context, authCtx RequestAuthContext, codeStr string) (*RedeemCodeResponse, error) {
	// 1. 验证并获取码
	code, err := s.validateAndGetCode(ctx, authCtx, codeStr)
	if err != nil {
		return nil, err
	}

	// 2. 执行兑换（带重试）
	redeemedCode, err := s.attemptRedeemWithRetry(ctx, authCtx, code, codeStr)
	if err != nil {
		return nil, err
	}

	// 3. 记录交易
	wallet, err := s.recordRedemptionTransaction(ctx, authCtx, redeemedCode, codeStr)
	if err != nil {
		return nil, err
	}

	return &RedeemCodeResponse{
		Success:    true,
		Amount:     redeemedCode.Amount,
		NewBalance: wallet.Balance,
		Message:    fmt.Sprintf("恭喜！你获得了 %d 積分", redeemedCode.Amount),
	}, nil
}

// GetCampaignStats 获取活动的统计数据
func (s *RedemptionCodeService) GetCampaignStats(ctx context.Context, authCtx RequestAuthContext, campaignID string) (*domain.CampaignStats, error) {
	// 检查权限
	if authCtx.Role != "owner" && authCtx.Role != "admin" {
		return nil, ErrUnauthorized
	}

	// 验证活动存在且属于该组织
	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	if campaign.OrganizationID != authCtx.OrganizationID {
		return nil, errors.New("campaign not in this organization")
	}

	// 获取统计
	return s.repo.GetCampaignStats(ctx, campaignID)
}

// CreateCampaignRequest 创建赎回活动的请求
type CreateCampaignRequest struct {
	Name        string // 活动名称
	Description string // 活动描述
}

// CreateCampaign 创建新的赎回活动
func (s *RedemptionCodeService) CreateCampaign(ctx context.Context, authCtx RequestAuthContext, req CreateCampaignRequest) (*domain.RedemptionCampaign, error) {
	// 检查权限
	if authCtx.Role != "owner" && authCtx.Role != "admin" {
		return nil, ErrUnauthorized
	}

	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("campaign name cannot be empty")
	}

	campaignID, err := domain.NewID()
	if err != nil {
		return nil, fmt.Errorf("generate campaign id: %w", err)
	}

	campaign := &domain.RedemptionCampaign{
		ID:             campaignID,
		Name:           req.Name,
		Description:    req.Description,
		Status:         domain.RedemptionCampaignStatusActive,
		CreatedBy:      authCtx.UserID,
		CreatedAt:      time.Now().UTC(),
		OrganizationID: authCtx.OrganizationID,
	}

	if err := s.repo.CreateCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	return campaign, nil
}

// generateRandomCode 生成随机的赎回码（8 个字符）
// 格式：GIFT-XXXXX（13 个字符总长度）
func generateRandomCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength := 8
	codeBytes := make([]byte, codeLength)

	for i := 0; i < codeLength; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		codeBytes[i] = charset[num.Int64()]
	}

	return fmt.Sprintf("GIFT-%s", string(codeBytes))
}
