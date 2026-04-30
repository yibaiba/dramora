package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type ProviderService struct {
	configs repo.ProviderConfigRepository
}

func NewProviderService(configs repo.ProviderConfigRepository) *ProviderService {
	return &ProviderService{configs: configs}
}

func (s *ProviderService) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	return s.configs.ListProviderConfigs(ctx)
}

func (s *ProviderService) GetProviderConfig(ctx context.Context, capability string) (domain.ProviderConfig, error) {
	return s.configs.GetProviderConfig(ctx, capability)
}

type SaveProviderConfigInput struct {
	Capability     string
	BaseURL        string
	APIKey         string
	Model          string
	CreditsPerUnit int
	CreditUnit     string
	TimeoutMS      int
	MaxRetries     int
}

func (s *ProviderService) SaveProviderConfig(ctx context.Context, input SaveProviderConfigInput) (domain.ProviderConfig, error) {
	if input.Capability == "" || input.BaseURL == "" || input.APIKey == "" || input.Model == "" {
		return domain.ProviderConfig{}, fmt.Errorf("%w: capability, base_url, api_key, model required", domain.ErrInvalidInput)
	}
	validCaps := map[string]bool{"chat": true, "image": true, "video": true, "audio": true}
	if !validCaps[input.Capability] {
		return domain.ProviderConfig{}, fmt.Errorf("%w: capability must be chat|image|video|audio", domain.ErrInvalidInput)
	}
	if input.CreditUnit == "" {
		input.CreditUnit = "per_call"
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = 120000
	}
	if input.MaxRetries <= 0 {
		input.MaxRetries = 3
	}

	id, err := domain.NewID()
	if err != nil {
		return domain.ProviderConfig{}, err
	}

	return s.configs.SaveProviderConfig(ctx, repo.SaveProviderConfigParams{
		ID:             id,
		Capability:     input.Capability,
		BaseURL:        input.BaseURL,
		APIKey:         input.APIKey,
		Model:          input.Model,
		CreditsPerUnit: input.CreditsPerUnit,
		CreditUnit:     input.CreditUnit,
		TimeoutMS:      input.TimeoutMS,
		MaxRetries:     input.MaxRetries,
	})
}

type TestProviderResult struct {
	OK        bool   `json:"ok"`
	Model     string `json:"model"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

func (s *ProviderService) TestProviderConfig(ctx context.Context, capability string) TestProviderResult {
	cfg, err := s.configs.GetProviderConfig(ctx, capability)
	if err != nil {
		return TestProviderResult{Error: "端点未配置"}
	}

	start := time.Now()
	client := &http.Client{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL+"/models", nil)
	if err != nil {
		return TestProviderResult{Error: fmt.Sprintf("构造请求失败: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return TestProviderResult{Error: fmt.Sprintf("连接失败: %v", err), LatencyMS: latency}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return TestProviderResult{Error: "API Key 无效 (401)", LatencyMS: latency}
	}
	if resp.StatusCode >= 400 {
		return TestProviderResult{Error: fmt.Sprintf("端点返回 %d", resp.StatusCode), LatencyMS: latency}
	}

	return TestProviderResult{OK: true, Model: cfg.Model, LatencyMS: latency}
}
