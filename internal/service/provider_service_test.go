package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type stubProviderRepo struct {
	saved repo.SaveProviderConfigParams
}

func (s *stubProviderRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}

func (s *stubProviderRepo) GetProviderConfig(_ context.Context, _ string) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, errors.New("not configured")
}

func (s *stubProviderRepo) SaveProviderConfig(_ context.Context, params repo.SaveProviderConfigParams) (domain.ProviderConfig, error) {
	s.saved = params
	return domain.ProviderConfig{
		ID:           params.ID,
		Capability:   params.Capability,
		ProviderType: params.ProviderType,
		BaseURL:      params.BaseURL,
		APIKey:       params.APIKey,
		Model:        params.Model,
	}, nil
}

func TestSaveProviderConfigAppliesCapabilityDefaults(t *testing.T) {
	cases := []struct {
		capability string
		expected   string
	}{
		{capability: "chat", expected: "openai"},
		{capability: "image", expected: "openai"},
		{capability: "video", expected: "seedance"},
		{capability: "audio", expected: "openai"},
	}
	for _, tc := range cases {
		t.Run(tc.capability, func(t *testing.T) {
			repo := &stubProviderRepo{}
			svc := NewProviderService(repo)
			cfg, err := svc.SaveProviderConfig(context.Background(), SaveProviderConfigInput{
				Capability: tc.capability,
				BaseURL:    "https://example.test",
				APIKey:     "k",
				Model:      "m",
			})
			if err != nil {
				t.Fatalf("save: %v", err)
			}
			if cfg.ProviderType != tc.expected {
				t.Fatalf("provider_type=%q want %q", cfg.ProviderType, tc.expected)
			}
		})
	}
}

func TestSaveProviderConfigRejectsWrongVendorPerCapability(t *testing.T) {
	cases := []struct {
		capability   string
		providerType string
	}{
		{capability: "chat", providerType: "seedance"},
		{capability: "image", providerType: "anthropic"},
		{capability: "video", providerType: "openai"},
		{capability: "audio", providerType: "seedance"},
	}
	for _, tc := range cases {
		t.Run(tc.capability+"_"+tc.providerType, func(t *testing.T) {
			svc := NewProviderService(&stubProviderRepo{})
			_, err := svc.SaveProviderConfig(context.Background(), SaveProviderConfigInput{
				Capability:   tc.capability,
				ProviderType: tc.providerType,
				BaseURL:      "https://example.test",
				APIKey:       "k",
				Model:        "m",
			})
			if err == nil {
				t.Fatalf("expected validation error for %s/%s", tc.capability, tc.providerType)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestSaveProviderConfigAcceptsAllowedCombinations(t *testing.T) {
	allowed := []struct {
		capability   string
		providerType string
	}{
		{capability: "chat", providerType: "anthropic"},
		{capability: "chat", providerType: "mock"},
		{capability: "image", providerType: "mock"},
		{capability: "video", providerType: "seedance"},
		{capability: "video", providerType: "mock"},
		{capability: "audio", providerType: "openai"},
	}
	for _, tc := range allowed {
		t.Run(tc.capability+"_"+tc.providerType, func(t *testing.T) {
			svc := NewProviderService(&stubProviderRepo{})
			if _, err := svc.SaveProviderConfig(context.Background(), SaveProviderConfigInput{
				Capability:   tc.capability,
				ProviderType: tc.providerType,
				BaseURL:      "https://example.test",
				APIKey:       "k",
				Model:        "m",
			}); err != nil {
				t.Fatalf("expected accept, got %v", err)
			}
		})
	}
}

// fixedConfigRepo always returns the same ProviderConfig regardless of capability key.
type fixedConfigRepo struct {
	cfg domain.ProviderConfig
}

func (r *fixedConfigRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return []domain.ProviderConfig{r.cfg}, nil
}
func (r *fixedConfigRepo) GetProviderConfig(_ context.Context, _ string) (domain.ProviderConfig, error) {
	return r.cfg, nil
}
func (r *fixedConfigRepo) SaveProviderConfig(_ context.Context, _ repo.SaveProviderConfigParams) (domain.ProviderConfig, error) {
	return r.cfg, nil
}

func TestCapabilityProbeURLMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		capability   string
		providerType string
		baseURL      string
		wantPresence bool
		wantURL      string
	}{
		{"chat", "openai", "https://api.openai.com/v1", false, "https://api.openai.com/v1/models"},
		{"chat", "anthropic", "https://api.anthropic.com", false, "https://api.anthropic.com/v1/models"},
		{"image", "openai", "https://api.openai.com/v1/", false, "https://api.openai.com/v1/models"},
		{"audio", "openai", "https://api.openai.com/v1", false, "https://api.openai.com/v1/models"},
		{"video", "seedance", "https://ark.example.com", true, ""},
		{"chat", "mock", "ignored", true, ""},
		{"image", "mock", "", true, ""},
	}
	for _, tc := range cases {
		got, presence := capabilityProbeURL(tc.capability, tc.providerType, tc.baseURL)
		if presence != tc.wantPresence {
			t.Fatalf("capability=%s vendor=%s presence=%v want %v", tc.capability, tc.providerType, presence, tc.wantPresence)
		}
		if got != tc.wantURL {
			t.Fatalf("capability=%s vendor=%s url=%q want %q", tc.capability, tc.providerType, got, tc.wantURL)
		}
	}
}

func TestTestProviderConfigRejectsCapabilityVendorMismatch(t *testing.T) {
	t.Parallel()

	svc := NewProviderService(&fixedConfigRepo{cfg: domain.ProviderConfig{
		Capability:   "image",
		ProviderType: "anthropic",
		BaseURL:      "https://api.anthropic.com",
		APIKey:       "k",
		Model:        "claude",
	}})
	res := svc.TestProviderConfig(context.Background(), "image")
	if res.OK {
		t.Fatalf("expected mismatch to fail, got %+v", res)
	}
	if res.Capability != "image" || res.ProviderType != "anthropic" {
		t.Fatalf("expected capability/provider_type echoed, got %+v", res)
	}
}

func TestTestProviderConfigMockReportsPresenceOnly(t *testing.T) {
	t.Parallel()

	svc := NewProviderService(&fixedConfigRepo{cfg: domain.ProviderConfig{
		Capability:   "image",
		ProviderType: "mock",
		Model:        "mock-image",
	}})
	res := svc.TestProviderConfig(context.Background(), "image")
	if !res.OK {
		t.Fatalf("expected mock to report OK, got %+v", res)
	}
	if res.Probe != "presence-only" {
		t.Fatalf("expected probe=presence-only, got %q", res.Probe)
	}
	if res.Capability != "image" || res.ProviderType != "mock" {
		t.Fatalf("expected capability/provider_type echoed, got %+v", res)
	}
}

func TestTestProviderConfigSeedanceRequiresCredentials(t *testing.T) {
	t.Parallel()

	svc := NewProviderService(&fixedConfigRepo{cfg: domain.ProviderConfig{
		Capability:   "video",
		ProviderType: "seedance",
		BaseURL:      "",
		APIKey:       "",
	}})
	res := svc.TestProviderConfig(context.Background(), "video")
	if res.OK || res.Error == "" {
		t.Fatalf("expected presence check to fail without creds, got %+v", res)
	}
	svc = NewProviderService(&fixedConfigRepo{cfg: domain.ProviderConfig{
		Capability:   "video",
		ProviderType: "seedance",
		BaseURL:      "https://ark.example.com",
		APIKey:       "k",
		Model:        "seedance-1",
	}})
	res = svc.TestProviderConfig(context.Background(), "video")
	if !res.OK || res.Probe != "presence-only" {
		t.Fatalf("expected seedance presence-only OK, got %+v", res)
	}
}
