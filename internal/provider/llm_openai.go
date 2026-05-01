package provider

import (
	"context"
	"time"
)

// openaiLLM wraps an OpenAI-compatible /chat/completions endpoint.
// Most commercial OpenAI-compatible gateways (DeepSeek, Moonshot,
// vLLM, OpenRouter, ...) work out of the box.
type openaiLLM struct {
	client *ChatClient
}

func newOpenAILLM(cfg LLMConfig) *openaiLLM {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &openaiLLM{
		client: NewChatClient(cfg.BaseURL, cfg.APIKey, cfg.Model, timeout),
	}
}

func (p *openaiLLM) Name() string { return "openai" }

func (p *openaiLLM) Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	resp, err := p.client.Complete(ctx, req.Messages)
	if err != nil {
		return nil, err
	}
	return &LLMResponse{
		Content:          resp.Content(),
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		Raw:              resp.Content(),
	}, nil
}
