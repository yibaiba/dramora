package provider

import (
	"fmt"
	"strings"
)

// NewImageProvider 按 cfg.ProviderType 路由 image capability。
// 支持矩阵：openai | mock。空值默认 openai（与 SaveProviderConfig 默认一致）。
func NewImageProvider(cfg CapabilityConfig) (ImageProvider, error) {
	switch normalizeProviderType(cfg.ProviderType, "openai") {
	case "openai":
		return newOpenAIImage(cfg), nil
	case "mock":
		return newMockImage(cfg), nil
	default:
		return nil, fmt.Errorf("unknown image provider_type %q (expected openai|mock)", cfg.ProviderType)
	}
}

// NewVideoProvider 按 cfg.ProviderType 路由 video capability。
// 支持矩阵：seedance | mock。空值默认 seedance。
func NewVideoProvider(cfg CapabilityConfig) (VideoProvider, error) {
	switch normalizeProviderType(cfg.ProviderType, "seedance") {
	case "seedance":
		return newSeedanceVideo(cfg), nil
	case "mock":
		return newMockVideo(cfg), nil
	default:
		return nil, fmt.Errorf("unknown video provider_type %q (expected seedance|mock)", cfg.ProviderType)
	}
}

// NewAudioProvider 按 cfg.ProviderType 路由 audio capability。
// 支持矩阵：openai | mock。空值默认 openai。
func NewAudioProvider(cfg CapabilityConfig) (AudioProvider, error) {
	switch normalizeProviderType(cfg.ProviderType, "openai") {
	case "openai":
		return newOpenAIAudio(cfg), nil
	case "mock":
		return newMockAudio(cfg), nil
	default:
		return nil, fmt.Errorf("unknown audio provider_type %q (expected openai|mock)", cfg.ProviderType)
	}
}

func normalizeProviderType(raw, fallback string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return fallback
	}
	return v
}
