package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
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
	ProviderType   string
	BaseURL        string
	APIKey         string
	Model          string
	CreditsPerUnit int
	CreditUnit     string
	TimeoutMS      int
	MaxRetries     int
}

// ValidProviderTypes lists provider adapter implementations recognised by the
// AgentService / NewLLMProvider factory. New entries must be added in the
// provider package as well.
var ValidProviderTypes = map[string]bool{
	"openai":    true,
	"anthropic": true,
	"mock":      true,
	"seedance":  true,
}

// CapabilityProviderTypes constrains which provider_type values are
// allowed per capability. The matrix reflects which adapters are
// actually implemented today: chat routes through the LLMProvider
// factory; image/audio support OpenAI-compatible endpoints (DALL-E /
// TTS-1 style) and offline mock; video accepts the Seedance ARK
// adapter and offline mock. New vendor adapters must register here
// before the admin UI exposes them.
var CapabilityProviderTypes = map[string]map[string]bool{
	"chat":  {"openai": true, "anthropic": true, "mock": true},
	"image": {"openai": true, "mock": true},
	"video": {"seedance": true, "mock": true},
	"audio": {"openai": true, "mock": true},
}

// CapabilityDefaultProviderType is the provider_type assigned when a
// caller saves a config without specifying provider_type. The value
// must exist in CapabilityProviderTypes[capability].
var CapabilityDefaultProviderType = map[string]string{
	"chat":  "openai",
	"image": "openai",
	"video": "seedance",
	"audio": "openai",
}

func allowedProviderTypeList(capability string) string {
	allowed, ok := CapabilityProviderTypes[capability]
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(allowed))
	for k := range allowed {
		parts = append(parts, k)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func (s *ProviderService) SaveProviderConfig(ctx context.Context, input SaveProviderConfigInput) (domain.ProviderConfig, error) {
	if input.Capability == "" || input.BaseURL == "" || input.APIKey == "" || input.Model == "" {
		return domain.ProviderConfig{}, fmt.Errorf("%w: capability, base_url, api_key, model required", domain.ErrInvalidInput)
	}
	allowed, ok := CapabilityProviderTypes[input.Capability]
	if !ok {
		return domain.ProviderConfig{}, fmt.Errorf("%w: capability must be chat|image|video|audio", domain.ErrInvalidInput)
	}
	if input.ProviderType == "" {
		input.ProviderType = CapabilityDefaultProviderType[input.Capability]
	}
	if !allowed[input.ProviderType] {
		return domain.ProviderConfig{}, fmt.Errorf(
			"%w: provider_type %q not allowed for capability %q (expected %s)",
			domain.ErrInvalidInput, input.ProviderType, input.Capability, allowedProviderTypeList(input.Capability),
		)
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
		ProviderType:   input.ProviderType,
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
	OK           bool   `json:"ok"`
	Capability   string `json:"capability,omitempty"`
	ProviderType string `json:"provider_type,omitempty"`
	Model        string `json:"model"`
	Probe        string `json:"probe,omitempty"`
	LatencyMS    int64  `json:"latency_ms"`
	Error        string `json:"error,omitempty"`
}

// capabilityProbeURL picks the most appropriate health-check endpoint for a
// given (capability, providerType) pair. We deliberately keep these probes
// capability-agnostic where possible (e.g. OpenAI's /v1/models authenticates
// the API key for chat / image / audio alike), but expose the chosen URL on
// the result so the admin UI can show what was actually probed.
//
// Returns ("", true) when the capability/vendor combination is presence-only
// and should skip the network round-trip (currently: mock for any capability,
// seedance video which is POST-only).
func capabilityProbeURL(capability, providerType, baseURL string) (probeURL string, presenceOnly bool) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	switch providerType {
	case "mock":
		return "", true
	case "seedance":
		return "", true
	case "anthropic":
		return base + "/v1/models", false
	case "openai":
		return base + "/models", false
	default:
		if base == "" {
			return "", false
		}
		return base + "/models", false
	}
}

func (s *ProviderService) TestProviderConfig(ctx context.Context, capability string) TestProviderResult {
	cfg, err := s.configs.GetProviderConfig(ctx, capability)
	if err != nil {
		return TestProviderResult{Capability: capability, Error: "端点未配置"}
	}

	resolvedType := cfg.ResolvedProviderType()
	allowed, ok := CapabilityProviderTypes[capability]
	if ok {
		if !allowed[resolvedType] {
			return TestProviderResult{
				Capability:   capability,
				ProviderType: resolvedType,
				Model:        cfg.Model,
				Error: fmt.Sprintf(
					"provider_type %q 与 capability %q 不匹配（允许：%s）",
					resolvedType, capability, allowedProviderTypeList(capability),
				),
			}
		}
	}

	probeURL, presenceOnly := capabilityProbeURL(capability, resolvedType, cfg.BaseURL)
	if presenceOnly {
		// Mock paths never need credentials; seedance-style POST-only paths
		// can only be sanity-checked by ensuring base_url + api_key exist.
		if resolvedType != "mock" {
			if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.BaseURL) == "" {
				return TestProviderResult{
					Capability:   capability,
					ProviderType: resolvedType,
					Model:        cfg.Model,
					Error:        "缺少 base_url 或 api_key",
				}
			}
		}
		return TestProviderResult{
			OK:           true,
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			Probe:        "presence-only",
		}
	}

	start := time.Now()
	client := &http.Client{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return TestProviderResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			Probe:        probeURL,
			Error:        fmt.Sprintf("构造请求失败: %v", err),
		}
	}
	if resolvedType == "anthropic" {
		req.Header.Set("x-api-key", cfg.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return TestProviderResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			Probe:        probeURL,
			Error:        fmt.Sprintf("连接失败: %v", err),
			LatencyMS:    latency,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return TestProviderResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			Probe:        probeURL,
			Error:        "API Key 无效 (401)",
			LatencyMS:    latency,
		}
	}
	if resp.StatusCode >= 400 {
		return TestProviderResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			Probe:        probeURL,
			Error:        fmt.Sprintf("端点返回 %d", resp.StatusCode),
			LatencyMS:    latency,
		}
	}

	return TestProviderResult{
		OK:           true,
		Capability:   capability,
		ProviderType: resolvedType,
		Model:        cfg.Model,
		Probe:        probeURL,
		LatencyMS:    latency,
	}
}
