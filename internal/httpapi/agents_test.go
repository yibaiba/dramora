package httpapi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type stubProviderRepo struct {
	cfg domain.ProviderConfig
}

func (r *stubProviderRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return []domain.ProviderConfig{r.cfg}, nil
}

func (r *stubProviderRepo) GetProviderConfig(_ context.Context, capability string) (domain.ProviderConfig, error) {
	if capability != r.cfg.Capability {
		return domain.ProviderConfig{}, fmt.Errorf("not found")
	}
	return r.cfg, nil
}

func (r *stubProviderRepo) SaveProviderConfig(_ context.Context, _ repo.SaveProviderConfigParams) (domain.ProviderConfig, error) {
	return r.cfg, nil
}

func TestStreamAgentRunEmitsDeltaAndDone(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("flusher unsupported")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		frames := []string{
			`{"choices":[{"delta":{"content":"Hello "}}]}`,
			`{"choices":[{"delta":{"content":"World"}}]}`,
			`{"usage":{"total_tokens":5}}`,
		}
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	stub := &stubProviderRepo{cfg: domain.ProviderConfig{
		ID:           "p1",
		Capability:   "chat",
		ProviderType: "openai",
		BaseURL:      upstream.URL,
		APIKey:       "test-key",
		Model:        "gpt-test",
		IsEnabled:    true,
	}}
	providerSvc := service.NewProviderService(stub)
	agentSvc := service.NewAgentService(providerSvc)

	router := NewRouter(RouterConfig{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:      "test",
		AgentService: agentSvc,
	})

	body := strings.NewReader(`{"role":"story_analyst","source_text":"小镇里的雨夜"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}

	var events []string
	scanner := bufio.NewScanner(rec.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events = append(events, strings.TrimPrefix(line, "event: "))
		}
	}

	deltaCount := 0
	hasDone := false
	for _, e := range events {
		switch e {
		case "delta":
			deltaCount++
		case "done":
			hasDone = true
		}
	}
	if deltaCount < 2 {
		t.Fatalf("expected >=2 delta events, got %d (%v)", deltaCount, events)
	}
	if !hasDone {
		t.Fatalf("expected done event, got %v", events)
	}
}

func TestStreamAgentRunRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	stub := &stubProviderRepo{cfg: domain.ProviderConfig{
		Capability: "chat",
		BaseURL:    "http://unused",
		APIKey:     "k",
		Model:      "m",
	}}
	providerSvc := service.NewProviderService(stub)
	agentSvc := service.NewAgentService(providerSvc)
	router := NewRouter(RouterConfig{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:      "test",
		AgentService: agentSvc,
	})

	cases := []struct {
		name string
		body string
	}{
		{"missing role", `{"source_text":"x"}`},
		{"missing source", `{"role":"story_analyst"}`},
		{"invalid json", `{not json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/stream", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestStreamAgentRunRequiresAgentService(t *testing.T) {
	t.Parallel()

	router := NewRouter(RouterConfig{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version: "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/stream",
		strings.NewReader(`{"role":"story_analyst","source_text":"x"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}
