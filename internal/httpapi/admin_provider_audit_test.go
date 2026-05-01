package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestProviderAuditCapturesSaveAndTest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)
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
		"provider_type": "openai",
		"base_url":      "https://example.com",
		"api_key":       "sk-test",
		"model":         "gpt-4o-mini",
	})
	saveResp := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers:save", bytes.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(saveResp, saveReq)
	if saveResp.Code != http.StatusOK {
		t.Fatalf("expected save 200, got %d: %s", saveResp.Code, saveResp.Body.String())
	}

	testResp := httptest.NewRecorder()
	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/chat:test", nil)
	router.ServeHTTP(testResp, testReq)
	if testResp.Code != http.StatusOK {
		t.Fatalf("expected test 200, got %d: %s", testResp.Code, testResp.Body.String())
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-audit", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected audit list 200, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var payload struct {
		Events  []providerAuditEventDTO `json:"events"`
		HasMore bool                    `json:"has_more"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Events) < 2 {
		t.Fatalf("expected at least 2 audit events (save+test), got %d", len(payload.Events))
	}
	actions := map[string]bool{}
	for _, ev := range payload.Events {
		actions[ev.Action] = true
		if ev.Capability != "chat" {
			t.Fatalf("expected capability chat, got %q", ev.Capability)
		}
		if ev.OrganizationID == "" {
			t.Fatalf("expected organization_id to be populated, got empty")
		}
	}
	if !actions["save"] || !actions["test"] {
		t.Fatalf("expected both save and test events, got actions=%v", actions)
	}

	csvResp := httptest.NewRecorder()
	csvReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-audit?format=csv", nil)
	router.ServeHTTP(csvResp, csvReq)
	if csvResp.Code != http.StatusOK {
		t.Fatalf("expected csv export 200, got %d: %s", csvResp.Code, csvResp.Body.String())
	}
	if ct := csvResp.Header().Get("Content-Type"); !bytes.Contains([]byte(ct), []byte("text/csv")) {
		t.Fatalf("expected text/csv content type, got %q", ct)
	}
	body := csvResp.Body.String()
	if !bytes.Contains([]byte(body), []byte("id,organization_id,action")) {
		t.Fatalf("expected csv header in body, got %q", body)
	}
	if !bytes.Contains([]byte(body), []byte(",save,")) || !bytes.Contains([]byte(body), []byte(",test,")) {
		t.Fatalf("expected save and test rows in csv, got %q", body)
	}
}

func TestProviderAuditRejectsViewerRole(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	const orgID = "00000000-0000-0000-0000-000000000001"
	authService := service.NewAuthService(identityRepo, "test-secret", nil)
	providerSvc := service.NewProviderService(repo.NewMemoryProviderConfigRepository())
	providerSvc.SetAuditRepository(repo.NewMemoryProviderAuditRepository())

	router := NewRouter(RouterConfig{
		Logger:          logger,
		Version:         "test",
		AuthService:     authService,
		ProviderService: providerSvc,
	})

	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         "00000000-0000-0000-0000-0000000000aa",
		OrganizationID: orgID,
		Email:          "viewer@example.com",
		DisplayName:    "Viewer",
		PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
		Role:           "viewer",
	})
	if err != nil {
		t.Fatalf("seed viewer: %v", err)
	}
	session, err := authService.IssueSessionForIdentity(identity)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-audit", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestProviderSaveRequiresOwnerRole(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	const orgID = "00000000-0000-0000-0000-000000000001"
	authService := service.NewAuthService(identityRepo, "test-secret", nil)
	providerSvc := service.NewProviderService(repo.NewMemoryProviderConfigRepository())
	providerSvc.SetAuditRepository(repo.NewMemoryProviderAuditRepository())

	router := NewRouter(RouterConfig{
		Logger:          logger,
		Version:         "test",
		AuthService:     authService,
		ProviderService: providerSvc,
	})

	// admin (read-only) seed
	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         "00000000-0000-0000-0000-0000000000ad",
		OrganizationID: orgID,
		Email:          "admin-readonly@example.com",
		DisplayName:    "Admin",
		PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
		Role:           "admin",
	})
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	session, err := authService.IssueSessionForIdentity(identity)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	// admin can read provider list
	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers", nil)
	listReq.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected admin GET /admin/providers 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	// admin cannot save provider config
	saveBody, _ := json.Marshal(map[string]any{
		"capability":    "chat",
		"provider_type": "openai",
		"base_url":      "https://example.com",
		"api_key":       "sk-test",
		"model":         "gpt-4o-mini",
	})
	saveResp := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers:save", bytes.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(saveResp, saveReq)
	if saveResp.Code != http.StatusForbidden {
		t.Fatalf("expected admin POST :save 403, got %d: %s", saveResp.Code, saveResp.Body.String())
	}

	// admin cannot run :test
	testResp := httptest.NewRecorder()
	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/chat:test", nil)
	testReq.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(testResp, testReq)
	if testResp.Code != http.StatusForbidden {
		t.Fatalf("expected admin POST :test 403, got %d: %s", testResp.Code, testResp.Body.String())
	}
}
