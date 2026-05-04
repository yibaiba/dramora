package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

// TestGenerateCodesSuccess 测试成功生成赎回码
func TestGenerateCodesSuccess(t *testing.T) {
	// 使用内存仓库进行测试
	memRepo := &MemoryRedemptionCodeRepository{
		campaigns: make(map[string]*domain.RedemptionCampaign),
		codes:     make(map[string]*domain.RedemptionCode),
	}

	walletRepo := repo.NewMemoryWalletRepository()
	walletSvc := service.NewWalletService(walletRepo, nil)
	redemptionSvc := service.NewRedemptionCodeService(memRepo, walletSvc, walletRepo)

	api := &api{
		redemptionCodeService: redemptionSvc,
	}

	reqBody := generateCodesRequest{
		Count:  10,
		Amount: 1000,
		Reason: "test reward",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/admin/redemption-codes:generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 添加认证上下文
	auth := service.RequestAuthContext{
		UserID:         "user-123",
		OrganizationID: "org-123",
		Role:           "admin",
	}
	ctx := service.WithRequestAuthContext(req.Context(), auth)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.generateCodes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("response: %s", w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	codes, ok := result["codes"].([]interface{})
	if !ok || len(codes) != 10 {
		t.Errorf("expected 10 codes, got %v", codes)
	}
}

// TestRedeemCodeSuccess 测试成功兑换赎回码
func TestRedeemCodeSuccess(t *testing.T) {
	memRepo := &MemoryRedemptionCodeRepository{
		campaigns: make(map[string]*domain.RedemptionCampaign),
		codes:     make(map[string]*domain.RedemptionCode),
	}

	walletRepo := repo.NewMemoryWalletRepository()
	walletSvc := service.NewWalletService(walletRepo, nil)

	// 初始化钱包
	walletRepo.ApplyTransaction(context.Background(), repo.WalletApplyParams{
		OrganizationID: "org-123",
		Kind:           domain.WalletKindCredit,
		Direction:      1,
		Amount:         0,
		Reason:         "init",
		ActorUserID:    "system",
	})

	redemptionSvc := service.NewRedemptionCodeService(memRepo, walletSvc, walletRepo)

	api := &api{
		redemptionCodeService: redemptionSvc,
	}

	// 首先生成一个赎回码
	auth := service.RequestAuthContext{
		UserID:         "admin-user",
		OrganizationID: "org-123",
		Role:           "admin",
	}

	generateReq := service.GenerateCodeRequest{
		Count:  1,
		Amount: 1000,
	}

	codes, err := redemptionSvc.GenerateCodes(context.Background(), auth, generateReq)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	if len(codes) == 0 {
		t.Fatal("no codes generated")
	}

	// 现在兑换码
	redeemReq := redeemCodeRequest{
		Code: codes[0],
	}

	body, _ := json.Marshal(redeemReq)
	req := httptest.NewRequest("POST", "/api/v1/redemption-codes:redeem", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 用户上下文
	userAuth := service.RequestAuthContext{
		UserID:         "user-123",
		OrganizationID: "org-123",
		Role:           "user",
	}
	ctx := service.WithRequestAuthContext(req.Context(), userAuth)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.redeemCode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("response: %s", w.Body.String())
	}

	var result service.RedeemCodeResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if !result.Success {
		t.Errorf("expected success, got %v", result.Success)
	}

	if result.Amount != 1000 {
		t.Errorf("expected amount 1000, got %d", result.Amount)
	}

	if result.NewBalance != 1000 {
		t.Errorf("expected new balance 1000, got %d", result.NewBalance)
	}
}

// TestRedeemCodeAlreadyUsed 测试兑换已使用的码
func TestRedeemCodeAlreadyUsed(t *testing.T) {
	memRepo := &MemoryRedemptionCodeRepository{
		campaigns: make(map[string]*domain.RedemptionCampaign),
		codes:     make(map[string]*domain.RedemptionCode),
	}

	walletRepo := repo.NewMemoryWalletRepository()
	walletSvc := service.NewWalletService(walletRepo, nil)
	redemptionSvc := service.NewRedemptionCodeService(memRepo, walletSvc, walletRepo)

	api := &api{
		redemptionCodeService: redemptionSvc,
	}

	// 生成码
	auth := service.RequestAuthContext{
		UserID:         "admin-user",
		OrganizationID: "org-123",
		Role:           "admin",
	}

	generateReq := service.GenerateCodeRequest{
		Count:  1,
		Amount: 1000,
	}

	codes, _ := redemptionSvc.GenerateCodes(context.Background(), auth, generateReq)
	code := codes[0]

	// 第一次兑换
	userAuth := service.RequestAuthContext{
		UserID:         "user-123",
		OrganizationID: "org-123",
		Role:           "user",
	}

	_, err := redemptionSvc.RedeemCode(context.Background(), userAuth, code)
	if err != nil {
		t.Fatalf("first redeem failed: %v", err)
	}

	// 第二次兑换应该失败
	redeemReq := redeemCodeRequest{
		Code: code,
	}

	body, _ := json.Marshal(redeemReq)
	req := httptest.NewRequest("POST", "/api/v1/redemption-codes:redeem", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := service.WithRequestAuthContext(req.Context(), userAuth)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.redeemCode(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var errorResp errorResponse
	json.Unmarshal(w.Body.Bytes(), &errorResp)

	if errorResp.Error.Code != "code_already_used" {
		t.Errorf("expected code_already_used, got %s", errorResp.Error.Code)
	}
}

// TestRedeemCodeExpired 测试兑换已过期的码
func TestRedeemCodeExpired(t *testing.T) {
	memRepo := &MemoryRedemptionCodeRepository{
		campaigns: make(map[string]*domain.RedemptionCampaign),
		codes:     make(map[string]*domain.RedemptionCode),
	}

	walletRepo := repo.NewMemoryWalletRepository()
	walletSvc := service.NewWalletService(walletRepo, nil)
	redemptionSvc := service.NewRedemptionCodeService(memRepo, walletSvc, walletRepo)

	api := &api{
		redemptionCodeService: redemptionSvc,
	}

	// 生成一个已过期的码
	auth := service.RequestAuthContext{
		UserID:         "admin-user",
		OrganizationID: "org-123",
		Role:           "admin",
	}

	pastTime := time.Now().UTC().Add(-24 * time.Hour)
	generateReq := service.GenerateCodeRequest{
		Count:     1,
		Amount:    1000,
		ExpiresAt: pastTime,
	}

	codes, _ := redemptionSvc.GenerateCodes(context.Background(), auth, generateReq)
	code := codes[0]

	// 尝试兑换
	userAuth := service.RequestAuthContext{
		UserID:         "user-123",
		OrganizationID: "org-123",
		Role:           "user",
	}

	redeemReq := redeemCodeRequest{
		Code: code,
	}

	body, _ := json.Marshal(redeemReq)
	req := httptest.NewRequest("POST", "/api/v1/redemption-codes:redeem", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := service.WithRequestAuthContext(req.Context(), userAuth)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.redeemCode(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var errorResp errorResponse
	json.Unmarshal(w.Body.Bytes(), &errorResp)

	if errorResp.Error.Code != "code_expired" {
		t.Errorf("expected code_expired, got %s", errorResp.Error.Code)
	}
}

// TestCampaignStats 测试获取活动统计数据
func TestCampaignStats(t *testing.T) {
	memRepo := &MemoryRedemptionCodeRepository{
		campaigns: make(map[string]*domain.RedemptionCampaign),
		codes:     make(map[string]*domain.RedemptionCode),
	}

	walletRepo := repo.NewMemoryWalletRepository()
	walletSvc := service.NewWalletService(walletRepo, nil)
	redemptionSvc := service.NewRedemptionCodeService(memRepo, walletSvc, walletRepo)

	api := &api{
		redemptionCodeService: redemptionSvc,
	}

	// 创建活动
	adminAuth := service.RequestAuthContext{
		UserID:         "admin-user",
		OrganizationID: "org-123",
		Role:           "admin",
	}

	campaign, _ := redemptionSvc.CreateCampaign(context.Background(), adminAuth, service.CreateCampaignRequest{
		Name: "Test Campaign",
	})

	// 生成码
	generateReq := service.GenerateCodeRequest{
		Count:      5,
		Amount:     1000,
		CampaignID: campaign.ID,
	}

	codes, _ := redemptionSvc.GenerateCodes(context.Background(), adminAuth, generateReq)

	// 兑换其中 2 个
	userAuth := service.RequestAuthContext{
		UserID:         "user-123",
		OrganizationID: "org-123",
		Role:           "user",
	}

	for i := 0; i < 2; i++ {
		redemptionSvc.RedeemCode(context.Background(), userAuth, codes[i])
	}

	// 查询统计
	req := httptest.NewRequest("GET", "/api/v1/admin/redemption-campaigns/"+campaign.ID+"/stats", nil)
	ctx := service.WithRequestAuthContext(req.Context(), adminAuth)
	req = req.WithContext(ctx)

	// 使用 chi 的 RouteContext 来设置路由参数
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("campaignId", campaign.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	api.getCampaignStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("response: %s", w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	stats := result["stats"].(map[string]interface{})
	if int(stats["total_codes"].(float64)) != 5 {
		t.Errorf("expected 5 total codes, got %v", stats["total_codes"])
	}

	if int(stats["used_codes"].(float64)) != 2 {
		t.Errorf("expected 2 used codes, got %v", stats["used_codes"])
	}
}

// MemoryRedemptionCodeRepository 内存实现用于测试
type MemoryRedemptionCodeRepository struct {
	campaigns map[string]*domain.RedemptionCampaign
	codes     map[string]*domain.RedemptionCode
}

func (r *MemoryRedemptionCodeRepository) CreateCampaign(ctx context.Context, campaign *domain.RedemptionCampaign) error {
	r.campaigns[campaign.ID] = campaign
	return nil
}

func (r *MemoryRedemptionCodeRepository) GetCampaignByID(ctx context.Context, campaignID string) (*domain.RedemptionCampaign, error) {
	if c, ok := r.campaigns[campaignID]; ok {
		return c, nil
	}
	return nil, domain.ErrCampaignNotFound
}

func (r *MemoryRedemptionCodeRepository) GenerateCodes(ctx context.Context, codes []*domain.RedemptionCode) error {
	for _, code := range codes {
		r.codes[code.Code] = code
	}
	return nil
}

func (r *MemoryRedemptionCodeRepository) GetCodeByCode(ctx context.Context, code string) (*domain.RedemptionCode, error) {
	// 查询时不区分大小写
	for _, c := range r.codes {
		if c.Code == code {
			return c, nil
		}
	}
	return nil, domain.ErrCodeNotFound
}

func (r *MemoryRedemptionCodeRepository) RedeemCode(ctx context.Context, code string, usedBy string, currentVersion int) (*domain.RedemptionCode, error) {
	for _, c := range r.codes {
		if c.Code == code {
			if c.Version != currentVersion {
				return nil, domain.ErrCodeVersionConflict
			}
			if c.Status != domain.RedemptionCodeStatusUnused {
				return nil, domain.ErrCodeAlreadyUsed
			}
			if c.IsExpired() {
				return nil, domain.ErrCodeExpired
			}

			c.UsedBy = usedBy
			c.UsedAt = time.Now().UTC()
			c.Status = domain.RedemptionCodeStatusUsed
			c.Version++
			return c, nil
		}
	}
	return nil, domain.ErrCodeNotFound
}

func (r *MemoryRedemptionCodeRepository) GetCampaignStats(ctx context.Context, campaignID string) (*domain.CampaignStats, error) {
	campaign, err := r.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	stats := &domain.CampaignStats{
		CampaignID: campaign.ID,
		Name:       campaign.Name,
	}

	for _, c := range r.codes {
		if c.CampaignID == campaignID {
			stats.TotalCodes++
			stats.TotalAmount += c.Amount

			if c.Status == domain.RedemptionCodeStatusUsed {
				stats.UsedCodes++
				stats.UsedAmount += c.Amount
			} else {
				stats.UnusedCodes++
				stats.UnusedAmount += c.Amount
			}
		}
	}

	if stats.TotalCodes > 0 {
		stats.UsagePercentage = float64(stats.UsedCodes) / float64(stats.TotalCodes) * 100
	}

	return stats, nil
}

func (r *MemoryRedemptionCodeRepository) ListCodes(ctx context.Context, filter repo.RedemptionCodeFilter) ([]*domain.RedemptionCode, error) {
	var result []*domain.RedemptionCode
	for _, c := range r.codes {
		if c.OrganizationID == filter.OrganizationID {
			if filter.Status == "" || c.Status == domain.RedemptionCodeStatus(filter.Status) {
				result = append(result, c)
			}
		}
	}
	return result, nil
}

func (r *MemoryRedemptionCodeRepository) ListCodesByCampaign(ctx context.Context, campaignID string, status string) ([]*domain.RedemptionCode, error) {
	var result []*domain.RedemptionCode
	for _, c := range r.codes {
		if c.CampaignID == campaignID {
			if status == "" || c.Status == domain.RedemptionCodeStatus(status) {
				result = append(result, c)
			}
		}
	}
	return result, nil
}
