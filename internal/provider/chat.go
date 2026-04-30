package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	ID      string               `json:"id"`
	Choices []ChatResponseChoice `json:"choices"`
	Usage   ChatUsage            `json:"usage"`
}

type ChatResponseChoice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (r *ChatResponse) Content() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

type ChatClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewChatClient(baseURL string, apiKey string, model string, timeout time.Duration) *ChatClient {
	return &ChatClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *ChatClient) Complete(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    c.model,
		Messages: messages,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat API returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	return &chatResp, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
