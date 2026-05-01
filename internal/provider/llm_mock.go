package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// mockLLM is a deterministic offline LLM adapter. It never performs any
// network IO. The reply is a JSON object echoing a short hash of the
// last user message, so downstream callers always receive parseable
// JSON. This is useful for local development, CI, and reproducible
// test runs against the agent service without provisioning real API
// credentials.
type mockLLM struct {
	model string
}

func newMockLLM(cfg LLMConfig) *mockLLM {
	model := cfg.Model
	if model == "" {
		model = "mock-llm"
	}
	return &mockLLM{model: model}
}

func (p *mockLLM) Name() string { return "mock" }

func (p *mockLLM) Complete(_ context.Context, req LLMRequest) (*LLMResponse, error) {
	var lastUser string
	for _, m := range req.Messages {
		if m.Role == "user" {
			lastUser = m.Content
		}
	}
	if lastUser == "" && len(req.Messages) > 0 {
		lastUser = req.Messages[len(req.Messages)-1].Content
	}

	digest := sha256.Sum256([]byte(lastUser))
	short := hex.EncodeToString(digest[:4])

	preview := strings.ReplaceAll(truncate(lastUser, 80), "\"", "'")
	preview = strings.ReplaceAll(preview, "\n", " ")

	body := fmt.Sprintf(
		`{"_mock":true,"model":%q,"echo_hash":%q,"echo_preview":%q}`,
		p.model, short, preview,
	)

	tokens := len(req.Messages)*4 + len(lastUser)/4
	return &LLMResponse{
		Content:          body,
		PromptTokens:     tokens,
		CompletionTokens: len(body) / 4,
		TotalTokens:      tokens + len(body)/4,
		Raw:              body,
	}, nil
}
