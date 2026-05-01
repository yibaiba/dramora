package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// openaiLLM wraps an OpenAI-compatible /chat/completions endpoint.
// Most commercial OpenAI-compatible gateways (DeepSeek, Moonshot,
// vLLM, OpenRouter, ...) work out of the box.
type openaiLLM struct {
	client *ChatClient
	apiKey string
	model  string
	http   *http.Client
}

func newOpenAILLM(cfg LLMConfig) *openaiLLM {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &openaiLLM{
		client: NewChatClient(cfg.BaseURL, cfg.APIKey, cfg.Model, timeout),
		apiKey: cfg.APIKey,
		model:  cfg.Model,
		// Streaming uses a separate client without a global timeout so
		// long Server-Sent-Events sessions are not aborted mid-stream.
		// Cancellation flows from ctx instead.
		http: &http.Client{},
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

type openaiStreamChoice struct {
	Index int `json:"index"`
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type openaiStreamFrame struct {
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *ChatUsage           `json:"usage,omitempty"`
}

func (p *openaiLLM) CompleteStream(ctx context.Context, req LLMRequest, onChunk StreamHandler) (*LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal openai stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.client.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build openai stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai stream API returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var sb strings.Builder
	var usage ChatUsage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var frame openaiStreamFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue // skip malformed keepalive frames
		}
		if frame.Usage != nil {
			usage = *frame.Usage
		}
		for _, ch := range frame.Choices {
			if ch.Delta.Content == "" {
				continue
			}
			sb.WriteString(ch.Delta.Content)
			if onChunk != nil {
				if err := onChunk(StreamChunk{Delta: ch.Delta.Content}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan openai stream: %w", err)
	}

	if onChunk != nil {
		if err := onChunk(StreamChunk{Done: true}); err != nil {
			return nil, err
		}
	}

	content := sb.String()
	return &LLMResponse{
		Content:          content,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		Raw:              content,
	}, nil
}
