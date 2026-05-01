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

// anthropicLLM speaks the Anthropic Messages API.
// See https://docs.anthropic.com/en/api/messages
//
// Differences from OpenAI:
//   - Endpoint is `/v1/messages` (relative to baseURL).
//   - System prompt is a top-level "system" string, not a message.
//   - Auth header is `x-api-key`, plus a required `anthropic-version`.
//   - Response payload places text in `content[0].text` and token
//     usage in `usage.input_tokens` / `usage.output_tokens`.
type anthropicLLM struct {
	baseURL    string
	apiKey     string
	model      string
	apiVersion string
	httpClient *http.Client
}

const defaultAnthropicVersion = "2023-06-01"
const defaultAnthropicMaxTokens = 4096

func newAnthropicLLM(cfg LLMConfig) *anthropicLLM {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &anthropicLLM{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		apiVersion: defaultAnthropicVersion,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (p *anthropicLLM) Name() string { return "anthropic" }

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicResponse struct {
	Content []anthropicResponseContent `json:"content"`
	Usage   anthropicUsage             `json:"usage"`
}

func (p *anthropicLLM) Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}

	system, messages := splitSystemPrompt(req.Messages)
	if len(messages) == 0 {
		return nil, fmt.Errorf("anthropic: at least one non-system message required")
	}

	body := anthropicRequest{
		Model:       model,
		System:      system,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", p.apiVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API returned %d: %s", resp.StatusCode, truncate(string(rawBody), 200))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	var sb strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	content := sb.String()

	return &LLMResponse{
		Content:          content,
		PromptTokens:     parsed.Usage.InputTokens,
		CompletionTokens: parsed.Usage.OutputTokens,
		TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		Raw:              string(rawBody),
	}, nil
}

type anthropicStreamFrame struct {
	Type  string `json:"type"`
	Delta struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		StopReason   string `json:"stop_reason,omitempty"`
		OutputTokens int    `json:"output_tokens,omitempty"`
	} `json:"delta"`
	Message struct {
		Usage anthropicUsage `json:"usage"`
	} `json:"message"`
	Usage anthropicUsage `json:"usage"`
}

func (p *anthropicLLM) CompleteStream(ctx context.Context, req LLMRequest, onChunk StreamHandler) (*LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}

	system, messages := splitSystemPrompt(req.Messages)
	if len(messages) == 0 {
		return nil, fmt.Errorf("anthropic: at least one non-system message required")
	}

	body := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
		"stream":     true,
	}
	if system != "" {
		body["system"] = system
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build anthropic stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", p.apiVersion)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Use a streaming-friendly client (no global timeout) so long
	// generations can run to completion. Cancellation flows via ctx.
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic stream API returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var sb strings.Builder
	var inputTokens, outputTokens int
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var frame anthropicStreamFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue
		}
		switch frame.Type {
		case "message_start":
			inputTokens = frame.Message.Usage.InputTokens
		case "content_block_delta":
			if frame.Delta.Type == "text_delta" && frame.Delta.Text != "" {
				sb.WriteString(frame.Delta.Text)
				if onChunk != nil {
					if err := onChunk(StreamChunk{Delta: frame.Delta.Text}); err != nil {
						return nil, err
					}
				}
			}
		case "message_delta":
			if frame.Usage.OutputTokens > 0 {
				outputTokens = frame.Usage.OutputTokens
			}
		case "message_stop":
			// terminal marker; loop exits naturally on EOF
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan anthropic stream: %w", err)
	}

	if onChunk != nil {
		if err := onChunk(StreamChunk{Done: true}); err != nil {
			return nil, err
		}
	}

	content := sb.String()
	return &LLMResponse{
		Content:          content,
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      inputTokens + outputTokens,
		Raw:              content,
	}, nil
}

func splitSystemPrompt(messages []ChatMessage) (string, []anthropicMessage) {
	var system strings.Builder
	out := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			if system.Len() > 0 {
				system.WriteString("\n\n")
			}
			system.WriteString(m.Content)
			continue
		}
		out = append(out, anthropicMessage{Role: m.Role, Content: m.Content})
	}
	return system.String(), out
}
