package provider

import "context"

// LLMRequest is the provider-agnostic chat-completion request shape.
// Adapters translate this into their underlying API contract.
type LLMRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// LLMResponse is the provider-agnostic chat-completion response.
// Content is the text reply; token counts are optional and may be 0
// when the underlying provider does not report usage.
type LLMResponse struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	// Raw is the unparsed response body (best-effort, for debugging).
	Raw string
}

// LLMProvider abstracts a chat-style large language model adapter.
// Implementations must be safe for use across goroutines as long as
// each Complete call uses its own context.
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}
