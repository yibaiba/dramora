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

// StreamChunk is a single delta emitted by CompleteStream. Delta holds
// the new text appended since the previous chunk; Done is true on the
// final chunk (which carries no additional Delta).
type StreamChunk struct {
	Delta string
	Done  bool
}

// StreamHandler receives chunks as they arrive. Returning a non-nil
// error aborts the stream and surfaces from CompleteStream.
type StreamHandler func(StreamChunk) error

// LLMProvider abstracts a chat-style large language model adapter.
// Implementations must be safe for use across goroutines as long as
// each Complete call uses its own context.
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error)
	// CompleteStream invokes the provider in streaming mode. Each
	// non-terminal chunk delivers an incremental Delta; a final
	// {Done:true} chunk is delivered before return. The aggregated
	// response is also returned so callers that only care about the
	// final body can ignore the handler.
	CompleteStream(ctx context.Context, req LLMRequest, onChunk StreamHandler) (*LLMResponse, error)
}
