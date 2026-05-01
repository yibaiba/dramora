package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewLLMProviderDefaultsToOpenAI(t *testing.T) {
	p, err := NewLLMProvider(LLMConfig{BaseURL: "http://x", APIKey: "k", Model: "m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "openai" {
		t.Fatalf("expected openai, got %s", p.Name())
	}
}

func TestNewLLMProviderUnknown(t *testing.T) {
	_, err := NewLLMProvider(LLMConfig{ProviderType: "weirdo"})
	if err == nil {
		t.Fatal("expected error for unknown provider type")
	}
	if !strings.Contains(err.Error(), "unknown provider_type") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestNewLLMProviderTypes(t *testing.T) {
	cases := map[string]string{
		"openai":     "openai",
		"OpenAI":     "openai",
		" anthropic": "anthropic",
		"mock":       "mock",
	}
	for input, want := range cases {
		p, err := NewLLMProvider(LLMConfig{ProviderType: input, BaseURL: "http://x", APIKey: "k", Model: "m"})
		if err != nil {
			t.Fatalf("type %q unexpected error: %v", input, err)
		}
		if p.Name() != want {
			t.Fatalf("type %q -> got %q want %q", input, p.Name(), want)
		}
	}
}

func TestMockLLMDeterministic(t *testing.T) {
	p, _ := NewLLMProvider(LLMConfig{ProviderType: "mock", Model: "fixture"})
	req := LLMRequest{Messages: []ChatMessage{
		{Role: "system", Content: "你是分析师"},
		{Role: "user", Content: "hello world"},
	}}
	a, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if a.Content != b.Content {
		t.Fatalf("mock not deterministic: %q vs %q", a.Content, b.Content)
	}
	// Must be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(a.Content), &parsed); err != nil {
		t.Fatalf("mock output not valid JSON: %v\n%s", err, a.Content)
	}
	if mock, _ := parsed["_mock"].(bool); !mock {
		t.Fatalf("expected _mock=true, got %v", parsed["_mock"])
	}
}

func TestOpenAILLMComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k1" {
			t.Errorf("unexpected auth: %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"gpt-test"`) {
			t.Errorf("model not propagated: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"x",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hi"}}],
			"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
		}`))
	}))
	defer srv.Close()

	p, err := NewLLMProvider(LLMConfig{ProviderType: "openai", BaseURL: srv.URL, APIKey: "k1", Model: "gpt-test", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), LLMRequest{Model: "gpt-test", Messages: []ChatMessage{{Role: "user", Content: "ping"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" {
		t.Fatalf("content=%q", resp.Content)
	}
	if resp.TotalTokens != 5 {
		t.Fatalf("tokens=%d", resp.TotalTokens)
	}
}

func TestAnthropicLLMComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "ak" {
			t.Errorf("x-api-key=%q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Errorf("missing anthropic-version header")
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if parsed["system"] != "你是助手" {
			t.Errorf("system not extracted: %v", parsed["system"])
		}
		msgs, _ := parsed["messages"].([]any)
		if len(msgs) != 1 {
			t.Errorf("expected 1 user msg, got %d", len(msgs))
		}
		if int(parsed["max_tokens"].(float64)) != defaultAnthropicMaxTokens {
			t.Errorf("max_tokens default not applied: %v", parsed["max_tokens"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content":[{"type":"text","text":"hello there"}],
			"usage":{"input_tokens":4,"output_tokens":3}
		}`))
	}))
	defer srv.Close()

	p, err := NewLLMProvider(LLMConfig{ProviderType: "anthropic", BaseURL: srv.URL, APIKey: "ak", Model: "claude-test", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), LLMRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "你是助手"},
			{Role: "user", Content: "ping"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello there" {
		t.Fatalf("content=%q", resp.Content)
	}
	if resp.PromptTokens != 4 || resp.CompletionTokens != 3 || resp.TotalTokens != 7 {
		t.Fatalf("usage mismatch: %+v", resp)
	}
}

func TestAnthropicRequiresNonSystemMessage(t *testing.T) {
	p, _ := NewLLMProvider(LLMConfig{ProviderType: "anthropic", BaseURL: "http://x", APIKey: "k", Model: "m"})
	_, err := p.Complete(context.Background(), LLMRequest{Messages: []ChatMessage{{Role: "system", Content: "only system"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "non-system message") {
		t.Fatalf("unexpected error: %v", err)
	}
}
