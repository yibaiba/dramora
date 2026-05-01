package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestSmokeChatProviderUsesMockAdapter(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret")
	providerCfgRepo := repo.NewMemoryProviderConfigRepository()
	auditRepo := repo.NewMemoryProviderAuditRepository()
	providerSvc := service.NewProviderService(providerCfgRepo)
	providerSvc.SetAuditRepository(auditRepo)

	rawRouter := NewRouter(RouterConfig{
		Logger:          logger,
		Version:         "test",
		AuthService:     authService,
		ProviderService: providerSvc,
	})
	router := newAuthenticatedTestRouter(rawRouter, authService)

	saveBody, _ := json.Marshal(map[string]any{
		"capability":    "chat",
		"provider_type": "mock",
		"base_url":      "https://example.com",
		"api_key":       "sk-test",
		"model":         "mock-1",
	})
	saveResp := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers:save", bytes.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(saveResp, saveReq)
	if saveResp.Code != http.StatusOK {
		t.Fatalf("expected save 200, got %d: %s", saveResp.Code, saveResp.Body.String())
	}

	smokeResp := httptest.NewRecorder()
	smokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/chat:smoke", nil)
	router.ServeHTTP(smokeResp, smokeReq)
	if smokeResp.Code != http.StatusOK {
		t.Fatalf("expected smoke 200, got %d: %s", smokeResp.Code, smokeResp.Body.String())
	}
	var payload struct {
		SmokeResult service.SmokeChatResult `json:"smoke_result"`
	}
	if err := json.NewDecoder(smokeResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.SmokeResult.OK {
		t.Fatalf("expected ok=true, got result=%+v", payload.SmokeResult)
	}
	if payload.SmokeResult.ProviderType != "mock" {
		t.Fatalf("expected provider_type=mock, got %q", payload.SmokeResult.ProviderType)
	}
	if payload.SmokeResult.Content == "" {
		t.Fatalf("expected non-empty content from mock provider, got empty")
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-audit?action=smoke", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected audit list 200, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var auditPayload struct {
		Events []providerAuditEventDTO `json:"events"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&auditPayload); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if len(auditPayload.Events) != 1 {
		t.Fatalf("expected exactly 1 smoke audit event, got %d", len(auditPayload.Events))
	}
	if !auditPayload.Events[0].Success {
		t.Fatalf("expected smoke audit success=true, got %+v", auditPayload.Events[0])
	}
}

func TestSmokeChatProviderReturnsErrorWhenChatNotConfigured(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret")
	providerCfgRepo := repo.NewMemoryProviderConfigRepository()
	providerSvc := service.NewProviderService(providerCfgRepo)

	rawRouter := NewRouter(RouterConfig{
		Logger:          logger,
		Version:         "test",
		AuthService:     authService,
		ProviderService: providerSvc,
	})
	router := newAuthenticatedTestRouter(rawRouter, authService)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/chat:smoke", nil)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 with error in body, got %d: %s", resp.Code, resp.Body.String())
	}
	var payload struct {
		SmokeResult service.SmokeChatResult `json:"smoke_result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.SmokeResult.OK {
		t.Fatalf("expected ok=false when chat not configured")
	}
	if payload.SmokeResult.Error == "" {
		t.Fatalf("expected error message, got empty")
	}
}
