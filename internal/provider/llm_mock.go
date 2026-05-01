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
	body, tokens := p.buildResponse(req)
	return &LLMResponse{
		Content:          body,
		PromptTokens:     tokens,
		CompletionTokens: len(body) / 4,
		TotalTokens:      tokens + len(body)/4,
		Raw:              body,
	}, nil
}

func (p *mockLLM) CompleteStream(ctx context.Context, req LLMRequest, onChunk StreamHandler) (*LLMResponse, error) {
	body, tokens := p.buildResponse(req)
	// Emit roughly 8-byte chunks to simulate token-by-token streaming
	// without making the test suite slow.
	const chunkSize = 8
	for i := 0; i < len(body); i += chunkSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := i + chunkSize
		if end > len(body) {
			end = len(body)
		}
		if onChunk != nil {
			if err := onChunk(StreamChunk{Delta: body[i:end]}); err != nil {
				return nil, err
			}
		}
	}
	if onChunk != nil {
		if err := onChunk(StreamChunk{Done: true}); err != nil {
			return nil, err
		}
	}
	return &LLMResponse{
		Content:          body,
		PromptTokens:     tokens,
		CompletionTokens: len(body) / 4,
		TotalTokens:      tokens + len(body)/4,
		Raw:              body,
	}, nil
}

func (p *mockLLM) buildResponse(req LLMRequest) (string, int) {
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
	return body, tokens
}
