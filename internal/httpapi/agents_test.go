package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/realtime"
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)

	router := NewRouter(RouterConfig{
		Logger:       logger,
		Version:      "test",
		AgentService: agentSvc,
		AuthService:  authService,
	})
	authenticatedRouter := newAuthenticatedTestRouter(router, authService)

	body := strings.NewReader(`{"role":"story_analyst","source_text":"小镇里的雨夜"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	authenticatedRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}
	runID := rec.Header().Get("X-Agent-Run-ID")
	if runID == "" {
		t.Fatal("expected X-Agent-Run-ID header")
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
	agentEventCount := 0
	for _, e := range events {
		switch e {
		case "delta":
			deltaCount++
		case "done":
			hasDone = true
		case "agent_event":
			agentEventCount++
		}
	}
	if deltaCount < 2 {
		t.Fatalf("expected >=2 delta events, got %d (%v)", deltaCount, events)
	}
	if !hasDone {
		t.Fatalf("expected done event, got %v", events)
	}
	if agentEventCount < 3 {
		t.Fatalf("expected >=3 agent_event frames, got %d (%v)", agentEventCount, events)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent-runs/"+runID, nil)
	statusRec := httptest.NewRecorder()
	authenticatedRouter.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", statusRec.Code, statusRec.Body.String())
	}
	var snapshot map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if snapshot["run_id"] != runID {
		t.Fatalf("expected run_id %q, got %#v", runID, snapshot["run_id"])
	}
	if snapshot["status"] != string(realtime.AgentRunStatusSucceeded) {
		t.Fatalf("expected succeeded status, got %#v", snapshot["status"])
	}
	if snapshot["latest_sequence"] != float64(3) {
		t.Fatalf("expected latest_sequence 3, got %#v", snapshot["latest_sequence"])
	}

	reconnectReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent-runs/"+runID+"/stream?after=1", nil)
	reconnectRec := httptest.NewRecorder()
	authenticatedRouter.ServeHTTP(reconnectRec, reconnectReq)
	if reconnectRec.Code != http.StatusOK {
		t.Fatalf("expected reconnect 200, got %d: %s", reconnectRec.Code, reconnectRec.Body.String())
	}
	if reconnectRec.Header().Get("X-Agent-Run-ID") != runID {
		t.Fatalf("expected reconnect header %q, got %q", runID, reconnectRec.Header().Get("X-Agent-Run-ID"))
	}
	reconnectBody := reconnectRec.Body.String()
	if !strings.Contains(reconnectBody, `"replay":true`) {
		t.Fatalf("expected replay frames, got %s", reconnectBody)
	}
	if !strings.Contains(reconnectBody, `"sequence":2`) || !strings.Contains(reconnectBody, `"sequence":3`) {
		t.Fatalf("expected replayed sequences 2 and 3, got %s", reconnectBody)
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
		{"invalid role", `{"role":"unknown_role","source_text":"x"}`},
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
