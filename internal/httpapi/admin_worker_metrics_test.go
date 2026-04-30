package httpapi

import (
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

func TestAdminWorkerMetricsRequiresAdminRole(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	const orgID = "00000000-0000-0000-0000-000000000001"
	authService := service.NewAuthService(identityRepo, "test-secret")
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProductionService: productionService,
	})

	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         "00000000-0000-0000-0000-0000000000bb",
		OrganizationID: orgID,
		Email:          "viewer-metrics@example.com",
		DisplayName:    "Viewer Metrics",
		PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
		Role:           "viewer",
	})
	if err != nil {
		t.Fatalf("seed viewer identity: %v", err)
	}
	session, err := authService.IssueSessionForIdentity(identity)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/worker-metrics", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer role, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAdminWorkerMetricsReturnsSnapshotForOwner(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	const orgID = "00000000-0000-0000-0000-000000000001"
	authService := service.NewAuthService(identityRepo, "test-secret")
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProductionService: productionService,
	})

	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         "00000000-0000-0000-0000-0000000000cc",
		OrganizationID: orgID,
		Email:          "owner-metrics@example.com",
		DisplayName:    "Owner Metrics",
		PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
		Role:           "owner",
	})
	if err != nil {
		t.Fatalf("seed owner identity: %v", err)
	}
	session, err := authService.IssueSessionForIdentity(identity)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/worker-metrics", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner role, got %d: %s", resp.Code, resp.Body.String())
	}

	var payload struct {
		WorkerMetrics workerMetricsDTO `json:"worker_metrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.WorkerMetrics.GenerationOrgUnresolvedSkips != 0 || payload.WorkerMetrics.ExportOrgUnresolvedSkips != 0 {
		t.Fatalf("expected zeroed counters initially, got %+v", payload.WorkerMetrics)
	}
}
