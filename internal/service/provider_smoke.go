package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider"
)

// SmokeChatResult 描述一次 chat 端到端探活结果。
// 与 TestProviderResult 不同：这里实际发起 1 次最小 LLM Complete 调用，
// 用以验证 base_url + api_key + model 真的能产出有效响应。
type SmokeChatResult struct {
	OK           bool   `json:"ok"`
	Capability   string `json:"capability"`
	ProviderType string `json:"provider_type"`
	Model        string `json:"model"`
	Content      string `json:"content,omitempty"`
	TokenCount   int    `json:"token_count,omitempty"`
	LatencyMS    int64  `json:"latency_ms"`
	Streamed     bool   `json:"streamed,omitempty"`
	ChunkCount   int    `json:"chunk_count,omitempty"`
	Error        string `json:"error,omitempty"`
}

const smokeChatPrompt = "Reply with the single word PONG."

// SmokeChatProvider 走完整的 LLMProvider.Complete 链路验证 chat 配置。
// mock 与真实 vendor 都会被实际触发，因此 mock=true 时也能拿到 deterministic 回复。
func (s *ProviderService) SmokeChatProvider(ctx context.Context) SmokeChatResult {
	const capability = "chat"
	cfg, err := s.configs.GetProviderConfig(ctx, capability)
	if err != nil {
		s.recordAudit(ctx, domain.ProviderAuditActionSmoke, capability, "", "", false, "端点未配置")
		return SmokeChatResult{Capability: capability, Error: "端点未配置"}
	}
	resolvedType := cfg.ResolvedProviderType()

	llm, err := provider.NewLLMProvider(provider.LLMConfig{
		ProviderType: resolvedType,
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Timeout:      time.Duration(cfg.TimeoutMS) * time.Millisecond,
	})
	if err != nil {
		msg := fmt.Sprintf("初始化 LLM 适配器失败: %v", err)
		s.recordAudit(ctx, domain.ProviderAuditActionSmoke, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{Capability: capability, ProviderType: resolvedType, Model: cfg.Model, Error: msg}
	}

	start := time.Now()
	resp, err := llm.Complete(ctx, provider.LLMRequest{
		Model: cfg.Model,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are a connectivity probe. Answer briefly."},
			{Role: "user", Content: smokeChatPrompt},
		},
		MaxTokens: 16,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := fmt.Sprintf("Complete 调用失败: %v", err)
		s.recordAudit(ctx, domain.ProviderAuditActionSmoke, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			LatencyMS:    latency,
			Error:        msg,
		}
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		msg := "provider 返回空响应"
		s.recordAudit(ctx, domain.ProviderAuditActionSmoke, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			LatencyMS:    latency,
			Error:        msg,
		}
	}
	preview := content
	if len(preview) > 240 {
		preview = preview[:240] + "…"
	}
	s.recordAudit(ctx, domain.ProviderAuditActionSmoke, capability, resolvedType, cfg.Model, true, fmt.Sprintf("latency=%dms", latency))
	return SmokeChatResult{
		OK:           true,
		Capability:   capability,
		ProviderType: resolvedType,
		Model:        cfg.Model,
		Content:      preview,
		TokenCount:   resp.TotalTokens,
		LatencyMS:    latency,
	}
}

// SmokeChatProviderStream 走 LLMProvider.CompleteStream 链路验证 chat 配置的流式输出能力。
// 在统计 chunk 数量的同时落 audit，验证 SSE / 流式解析全链路工作正常。
func (s *ProviderService) SmokeChatProviderStream(ctx context.Context) SmokeChatResult {
	const capability = "chat"
	cfg, err := s.configs.GetProviderConfig(ctx, capability)
	if err != nil {
		s.recordAudit(ctx, domain.ProviderAuditActionSmokeStream, capability, "", "", false, "端点未配置")
		return SmokeChatResult{Capability: capability, Streamed: true, Error: "端点未配置"}
	}
	resolvedType := cfg.ResolvedProviderType()

	llm, err := provider.NewLLMProvider(provider.LLMConfig{
		ProviderType: resolvedType,
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Timeout:      time.Duration(cfg.TimeoutMS) * time.Millisecond,
	})
	if err != nil {
		msg := fmt.Sprintf("初始化 LLM 适配器失败: %v", err)
		s.recordAudit(ctx, domain.ProviderAuditActionSmokeStream, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{Capability: capability, ProviderType: resolvedType, Model: cfg.Model, Streamed: true, Error: msg}
	}

	var (
		chunkCount int
		streamed   strings.Builder
	)
	start := time.Now()
	resp, err := llm.CompleteStream(ctx, provider.LLMRequest{
		Model: cfg.Model,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are a connectivity probe. Answer briefly."},
			{Role: "user", Content: smokeChatPrompt},
		},
		MaxTokens: 16,
	}, func(chunk provider.StreamChunk) error {
		if chunk.Done {
			return nil
		}
		if chunk.Delta != "" {
			chunkCount++
			streamed.WriteString(chunk.Delta)
		}
		return nil
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := fmt.Sprintf("CompleteStream 调用失败: %v", err)
		s.recordAudit(ctx, domain.ProviderAuditActionSmokeStream, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			LatencyMS:    latency,
			Streamed:     true,
			ChunkCount:   chunkCount,
			Error:        msg,
		}
	}
	content := strings.TrimSpace(streamed.String())
	if content == "" && resp != nil {
		content = strings.TrimSpace(resp.Content)
	}
	if content == "" {
		msg := "stream 未产生任何 delta"
		s.recordAudit(ctx, domain.ProviderAuditActionSmokeStream, capability, resolvedType, cfg.Model, false, msg)
		return SmokeChatResult{
			Capability:   capability,
			ProviderType: resolvedType,
			Model:        cfg.Model,
			LatencyMS:    latency,
			Streamed:     true,
			ChunkCount:   chunkCount,
			Error:        msg,
		}
	}
	preview := content
	if len(preview) > 240 {
		preview = preview[:240] + "…"
	}
	tokens := 0
	if resp != nil {
		tokens = resp.TotalTokens
	}
	s.recordAudit(
		ctx,
		domain.ProviderAuditActionSmokeStream,
		capability,
		resolvedType,
		cfg.Model,
		true,
		fmt.Sprintf("latency=%dms chunks=%d", latency, chunkCount),
	)
	return SmokeChatResult{
		OK:           true,
		Capability:   capability,
		ProviderType: resolvedType,
		Model:        cfg.Model,
		Content:      preview,
		TokenCount:   tokens,
		LatencyMS:    latency,
		Streamed:     true,
		ChunkCount:   chunkCount,
	}
}
