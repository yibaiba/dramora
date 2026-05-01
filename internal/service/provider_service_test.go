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
