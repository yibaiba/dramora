package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/workflow"
)

type AgentService struct {
	providerSvc      *ProviderService
	executorFactory  func(sourceText string) workflow.NodeExecutor
	availabilityFunc func(ctx context.Context) bool
	telemetry        *llmTelemetry
	registry         *AgentRegistry
}

func NewAgentService(providerSvc *ProviderService) *AgentService {
	return &AgentService{
		providerSvc: providerSvc,
		telemetry:   newLLMTelemetry(),
		registry:    mustNewAgentRegistry(),
	}
}

// LLMTelemetry returns a snapshot of the in-process LLM call telemetry.
// Safe to call from any goroutine; never returns nil even when the
// service was constructed without a telemetry buffer (legacy paths).
func (s *AgentService) LLMTelemetry() LLMTelemetrySnapshot {
	if s == nil || s.telemetry == nil {
		return LLMTelemetrySnapshot{
			ByVendor:            map[string]uint64{},
			ByCapability:        map[string]uint64{},
			AvgDurationMSVendor: map[string]int64{},
		}
	}
	return s.telemetry.snapshot()
}

// LLMTelemetryWindow returns a rolling N-day window aggregate. Returns nil
// when no persistent repository is wired (in-memory only deployments).
func (s *AgentService) LLMTelemetryWindow(ctx context.Context, days int) (*LLMTelemetryWindowSnapshot, error) {
	if s == nil || s.telemetry == nil {
		return nil, nil
	}
	return s.telemetry.WindowSnapshot(ctx, days)
}

// RecordTelemetry lets other services (e.g. ProductionService for image / audio /
// video worker calls) feed events into the same in-process telemetry buffer.
func (s *AgentService) RecordTelemetry(ev LLMTelemetryEvent) {
	if s == nil || s.telemetry == nil {
		return
	}
	s.telemetry.record(ev)
}

// SetTelemetryRepository wires a persistent backend so per-vendor / per-capability
// telemetry counters survive restarts. Safe to call once at startup.
func (s *AgentService) SetTelemetryRepository(r repo.LLMTelemetryRepository) {
	if s == nil || s.telemetry == nil {
		return
	}
	s.telemetry.SetRepository(r)
}

// HydrateTelemetry replays persisted aggregates into the in-memory counters.
func (s *AgentService) HydrateTelemetry(ctx context.Context) error {
	if s == nil || s.telemetry == nil {
		return nil
	}
	return s.telemetry.Hydrate(ctx)
}

// ResetTelemetry clears in-memory counters/recent events and persisted aggregates.
func (s *AgentService) ResetTelemetry(ctx context.Context) error {
	if s == nil || s.telemetry == nil {
		return nil
	}
	return s.telemetry.Reset(ctx)
}

func (s *AgentService) recordTelemetry(ev LLMTelemetryEvent) {
	if s == nil || s.telemetry == nil {
		return
	}
	s.telemetry.record(ev)
}

type AgentResult struct {
	Role        string
	Output      string
	Highlights  []string
	TokenCount  int
	DurationMS  int64
	RawResponse string
}

func (s *AgentService) MakeNodeExecutor(sourceText string) workflow.NodeExecutor {
	if s != nil && s.executorFactory != nil {
		return s.executorFactory(sourceText)
	}
	return func(ctx context.Context, nodeID string, kind workflow.NodeKind, bb *workflow.Blackboard) (any, error) {
		prompt, err := s.buildAgentPrompt(nodeID, sourceText, bb)
		if err != nil {
			return nil, err
		}
		result, err := s.callLLM(ctx, nodeID, prompt)
		if err != nil {
			return nil, err
		}
		bb.Write(nodeID, result)
		return result, nil
	}
}

func (s *AgentService) callLLM(ctx context.Context, role string, prompt string) (*AgentResult, error) {
	systemPrompt, err := s.systemPromptForRole(role)
	if err != nil {
		return nil, err
	}

	cfg, err := s.providerSvc.GetProviderConfig(ctx, "chat")
	if err != nil {
		return nil, fmt.Errorf("chat 端点未配置: %w", err)
	}

	llm, err := provider.NewLLMProvider(provider.LLMConfig{
		ProviderType: cfg.ResolvedProviderType(),
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Timeout:      time.Duration(cfg.TimeoutMS) * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 LLM 适配器失败: %w", err)
	}

	start := time.Now()
	resp, err := llm.Complete(ctx, provider.LLMRequest{
		Model: cfg.Model,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	elapsed := time.Since(start).Milliseconds()
	tokens := 0
	if resp != nil {
		tokens = resp.TotalTokens
	}
	ev := LLMTelemetryEvent{
		StartedAt:  start.UTC(),
		Capability: "chat",
		Vendor:     cfg.ResolvedProviderType(),
		Model:      cfg.Model,
		Role:       role,
		Mode:       "complete",
		DurationMS: elapsed,
		TokenCount: tokens,
		Success:    err == nil,
	}
	if err != nil {
		ev.ErrorMessage = err.Error()
	}
	s.recordTelemetry(ev)
	if err != nil {
		return nil, err
	}

	content := resp.Content
	highlights := extractHighlights(role, content)

	return &AgentResult{
		Role:        role,
		Output:      content,
		Highlights:  highlights,
		TokenCount:  resp.TotalTokens,
		DurationMS:  elapsed,
		RawResponse: content,
	}, nil
}

// RunSingleAgentStream 执行一次单 Agent 调用并通过 onDelta 回调流式输出。
// 与 callLLM 一致地复用 buildAgentPrompt + provider 抽象，但走 CompleteStream。
// contextMap 允许调用方为依赖前序 Agent 的 role（如 outline_planner / screenwriter）
// 注入已有产出，避免在 HTTP 层重新跑整个 workflow。
func (s *AgentService) RunSingleAgentStream(ctx context.Context, role string, sourceText string, contextMap map[string]string, onDelta func(string) error) (*AgentResult, error) {
	if role == "" {
		return nil, fmt.Errorf("role required")
	}
	systemPrompt, err := s.systemPromptForRole(role)
	if err != nil {
		return nil, err
	}

	cfg, err := s.providerSvc.GetProviderConfig(ctx, "chat")
	if err != nil {
		return nil, fmt.Errorf("chat 端点未配置: %w", err)
	}

	llm, err := provider.NewLLMProvider(provider.LLMConfig{
		ProviderType: cfg.ResolvedProviderType(),
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Timeout:      time.Duration(cfg.TimeoutMS) * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 LLM 适配器失败: %w", err)
	}

	bb := workflow.NewBlackboard()
	for k, v := range contextMap {
		bb.Write(k, &AgentResult{Role: k, Output: v, RawResponse: v})
	}
	prompt, err := s.buildAgentPrompt(role, sourceText, bb)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := llm.CompleteStream(ctx, provider.LLMRequest{
		Model: cfg.Model,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}, func(chunk provider.StreamChunk) error {
		if onDelta == nil || chunk.Delta == "" {
			return nil
		}
		return onDelta(chunk.Delta)
	})
	elapsed := time.Since(start).Milliseconds()
	tokens := 0
	if resp != nil {
		tokens = resp.TotalTokens
	}
	ev := LLMTelemetryEvent{
		StartedAt:  start.UTC(),
		Capability: "chat",
		Vendor:     cfg.ResolvedProviderType(),
		Model:      cfg.Model,
		Role:       role,
		Mode:       "stream",
		DurationMS: elapsed,
		TokenCount: tokens,
		Success:    err == nil,
	}
	if err != nil {
		ev.ErrorMessage = err.Error()
	}
	s.recordTelemetry(ev)
	if err != nil {
		return nil, err
	}

	content := resp.Content
	highlights := extractHighlights(role, content)

	return &AgentResult{
		Role:        role,
		Output:      content,
		Highlights:  highlights,
		TokenCount:  resp.TotalTokens,
		DurationMS:  elapsed,
		RawResponse: content,
	}, nil
}

func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if s != nil && s.availabilityFunc != nil {
		return s.availabilityFunc(ctx)
	}
	_, err := s.providerSvc.GetProviderConfig(ctx, "chat")
	return err == nil
}

func (s *AgentService) SupportsRole(role string) bool {
	return s != nil && s.registry != nil && s.registry.HasRole(role)
}

func (s *AgentService) buildAgentPrompt(role string, sourceText string, bb *workflow.Blackboard) (string, error) {
	if s == nil || s.registry == nil {
		return "", fmt.Errorf("agent registry not configured")
	}
	return s.registry.BuildUserPrompt(role, sourceText, bb)
}

func (s *AgentService) systemPromptForRole(role string) (string, error) {
	if s == nil || s.registry == nil {
		return "", fmt.Errorf("agent registry not configured")
	}
	return s.registry.SystemPrompt(role)
}

func extractHighlights(role string, content string) []string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return []string{truncateStr(content, 80)}
	}

	switch role {
	case "story_analyst":
		var data struct {
			Themes []string `json:"themes"`
		}
		json.Unmarshal([]byte(content), &data)
		return data.Themes
	case "outline_planner":
		var data struct {
			Beats []struct {
				Code  string `json:"code"`
				Title string `json:"title"`
			} `json:"beats"`
		}
		json.Unmarshal([]byte(content), &data)
		titles := make([]string, 0, len(data.Beats))
		for _, b := range data.Beats {
			titles = append(titles, b.Code+" "+b.Title)
		}
		return titles
	case "character_analyst":
		return extractNames(content, "characters")
	case "scene_analyst":
		return extractNames(content, "scenes")
	case "prop_analyst":
		return extractNames(content, "props")
	case "screenwriter":
		return extractSceneHighlights(content)
	case "director":
		return extractDirectorHighlights(content)
	case "cinematographer":
		return extractCinematographerHighlights(content)
	case "voice_subtitle":
		return extractVoiceHighlights(content)
	}
	return nil
}

func extractNames(content string, key string) []string {
	var data map[string][]struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	items := data[key]
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Code+" "+item.Name)
	}
	return names
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func extractSceneHighlights(content string) []string {
	var data struct {
		Scenes []struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.Scenes))
	for _, s := range data.Scenes {
		highlights = append(highlights, s.Code+" "+s.Title)
	}
	return highlights
}

func extractDirectorHighlights(content string) []string {
	var data struct {
		VisualPlan struct {
			ContinuityNotes []string `json:"continuity_notes"`
		} `json:"visual_plan"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	return data.VisualPlan.ContinuityNotes
}

func extractCinematographerHighlights(content string) []string {
	var data struct {
		Shots []struct {
			Scene    string `json:"scene"`
			ShotSize string `json:"shot_size"`
		} `json:"shots"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.Shots))
	for _, s := range data.Shots {
		highlights = append(highlights, s.Scene+" "+s.ShotSize)
	}
	return highlights
}

func extractVoiceHighlights(content string) []string {
	var data struct {
		VoiceSegments []struct {
			Character string `json:"character"`
			Style     string `json:"style"`
		} `json:"voice_segments"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.VoiceSegments))
	for _, v := range data.VoiceSegments {
		highlights = append(highlights, v.Character+" · "+v.Style)
	}
	return highlights
}
