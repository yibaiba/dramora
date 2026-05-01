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

func setupWalletRouter(t *testing.T) (*service.AuthService, *repo.MemoryIdentityRepository, http.Handler) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	authService := service.NewAuthService(identityRepo, "test-secret", nil)
	walletRepo := repo.NewMemoryWalletRepository()
	walletService := service.NewWalletService(walletRepo, nil)
	router := NewRouter(RouterConfig{
		Logger:        logger,
		Version:       "test",
		AuthService:   authService,
		WalletService: walletService,
	})
	return authService, identityRepo, router
}

func issueWalletSession(t *testing.T, authService *service.AuthService, identityRepo *repo.MemoryIdentityRepository, userID, orgID, email, role string) string {
	t.Helper()
	identity, err := identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		DisplayName:    "Wallet Test " + role,
		PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
		Role:           role,
	})
	if err != nil {
		t.Fatalf("seed identity %s: %v", role, err)
	}
	session, err := authService.IssueSessionForIdentity(identity)
	if err != nil {
		t.Fatalf("issue session %s: %v", role, err)
	}
	return session.Token
}

func TestWalletGetReturnsZeroBalanceForFreshOrg(t *testing.T) {
	t.Parallel()
	authService, identityRepo, router := setupWalletRouter(t)
	const orgID = "00000000-0000-0000-0000-000000000001"
	token := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000aa1", orgID, "owner-wallet-zero@example.com", "owner")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		Wallet struct {
			Balance int64 `json:"balance"`
		} `json:"wallet"`
		RecentTransactions []any `json:"recent_transactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Wallet.Balance != 0 {
		t.Fatalf("expected zero balance, got %d", body.Wallet.Balance)
	}
	if len(body.RecentTransactions) != 0 {
		t.Fatalf("expected empty tx list, got %d", len(body.RecentTransactions))
	}
}

func TestWalletCreditAndDebitFlowsUpdateBalance(t *testing.T) {
	t.Parallel()
	authService, identityRepo, router := setupWalletRouter(t)
	const orgID = "00000000-0000-0000-0000-000000000002"
	token := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000aa2", orgID, "owner-wallet-flow@example.com", "owner")

	body := bytes.NewBufferString(`{"amount":1000,"reason":"manual top-up","ref_type":"manual"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet:credit", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("credit expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	body = bytes.NewBufferString(`{"amount":300,"reason":"smoke run","ref_type":"generation_job","ref_id":"job-1"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/wallet:debit", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("debit expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("get expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var snapshot struct {
		Wallet struct {
			Balance int64 `json:"balance"`
		} `json:"wallet"`
		RecentTransactions []struct {
			Kind         string `json:"kind"`
			Amount       int64  `json:"amount"`
			BalanceAfter int64  `json:"balance_after"`
		} `json:"recent_transactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.Wallet.Balance != 700 {
		t.Fatalf("expected balance 700, got %d", snapshot.Wallet.Balance)
	}
	if len(snapshot.RecentTransactions) != 2 {
		t.Fatalf("expected 2 recent transactions, got %d", len(snapshot.RecentTransactions))
	}
	if snapshot.RecentTransactions[0].Kind != "debit" || snapshot.RecentTransactions[0].BalanceAfter != 700 {
		t.Fatalf("unexpected most recent tx: %+v", snapshot.RecentTransactions[0])
	}
}

func TestWalletDebitInsufficientBalanceReturns422(t *testing.T) {
	t.Parallel()
	authService, identityRepo, router := setupWalletRouter(t)
	const orgID = "00000000-0000-0000-0000-000000000003"
	token := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000aa3", orgID, "owner-wallet-overdraw@example.com", "owner")

	body := bytes.NewBufferString(`{"amount":50,"reason":"overdraw"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet:debit", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for insufficient balance, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestWalletCreditForbiddenForViewerRole(t *testing.T) {
	t.Parallel()
	authService, identityRepo, router := setupWalletRouter(t)
	const orgID = "00000000-0000-0000-0000-000000000004"
	token := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000aa4", orgID, "viewer-wallet@example.com", "viewer")

	body := bytes.NewBufferString(`{"amount":100,"reason":"viewer attempt"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet:credit", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer credit, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestWalletScopedToCallingOrganization(t *testing.T) {
	t.Parallel()
	authService, identityRepo, router := setupWalletRouter(t)
	tokenA := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000ab1",
		"00000000-0000-0000-0000-000000000005", "owner-a-wallet@example.com", "owner")
	tokenB := issueWalletSession(t, authService, identityRepo,
		"00000000-0000-0000-0000-000000000ab2",
		"00000000-0000-0000-0000-000000000006", "owner-b-wallet@example.com", "owner")

	body := bytes.NewBufferString(`{"amount":500,"reason":"top up org A"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet:credit", body)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("credit org A expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("get org B expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var snapshot struct {
		Wallet struct {
			Balance int64 `json:"balance"`
		} `json:"wallet"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snapshot.Wallet.Balance != 0 {
		t.Fatalf("expected org B balance 0, got %d", snapshot.Wallet.Balance)
	}
}
