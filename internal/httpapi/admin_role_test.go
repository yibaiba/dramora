package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestAdminRoutesRequireOwnerRole(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)
	router := NewRouter(RouterConfig{
		Logger:      logger,
		Version:     "test",
		AuthService: authService,
	})

	// Default Register grants "owner" — so we forge a token with a non-owner role
	// by going through the auth service and then re-signing isn't trivial; instead
	// we exercise the middleware by calling the admin endpoint without a token,
	// which should produce a 401 from the auth middleware.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers", nil)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d: %s", resp.Code, resp.Body.String())
	}

	// With an invalid token we still expect 401 from auth middleware
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAdminRoutesRejectNonAdminRole(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	const orgID = "00000000-0000-0000-0000-000000000001"
	authService := service.NewAuthService(identityRepo, "test-secret", nil)
	router := NewRouter(RouterConfig{
		Logger:      logger,
		Version:     "test",
		AuthService: authService,
	})

	// Pre-create a viewer-role user directly via the repo to bypass the
	// "always-owner" Register default and assert the role gate kicks in.
	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         "00000000-0000-0000-0000-0000000000aa",
		OrganizationID: orgID,
		Email:          "viewer@example.com",
		DisplayName:    "Viewer",
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer role, got %d: %s", resp.Code, resp.Body.String())
	}
}
