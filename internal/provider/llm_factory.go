package provider

import (
	"fmt"
	"strings"
	"time"
)

// LLMConfig is the provider-agnostic configuration consumed by
// NewLLMProvider. Keep this struct primitive-only so the provider
// package does not need to depend on the domain layer.
type LLMConfig struct {
	ProviderType string // openai | anthropic | mock (case-insensitive, defaults to openai)
	BaseURL      string
	APIKey       string
	Model        string
	Timeout      time.Duration
}

// NewLLMProvider returns the LLMProvider implementation matching
// cfg.ProviderType. Empty type defaults to "openai" so legacy
// configs without provider_type keep working.
func NewLLMProvider(cfg LLMConfig) (LLMProvider, error) {
	kind := strings.ToLower(strings.TrimSpace(cfg.ProviderType))
	if kind == "" {
		kind = "openai"
	}
	switch kind {
	case "openai":
		return newOpenAILLM(cfg), nil
	case "anthropic":
		return newAnthropicLLM(cfg), nil
	case "mock":
		return newMockLLM(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider_type %q (expected openai|anthropic|mock)", cfg.ProviderType)
	}
}
