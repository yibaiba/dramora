package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAccountChangePassword(t *testing.T) {
	t.Parallel()

	router := testRouter()

	changeResp := httptest.NewRecorder()
	changeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account:change-password",
		bytes.NewBufferString(`{"current_password":"strongpass","new_password":"strongpass-2"}`),
	)
	router.ServeHTTP(changeResp, changeReq)
	if changeResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 changing password, got %d: %s", changeResp.Code, changeResp.Body.String())
	}

	loginResp := httptest.NewRecorder()
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		body(`{"email":"test-director@example.com","password":"strongpass-2"}`),
	)
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login 200 with new password, got %d: %s", loginResp.Code, loginResp.Body.String())
	}
}

func TestAccountAPIKeysLifecycle(t *testing.T) {
	t.Parallel()

	router := testRouter()
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account/api-keys",
		bytes.NewBufferString(`{"name":"Studio automation","scope":"write","expires_at":"`+expiresAt+`"}`),
	)
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating api key, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		APIKey createdAccountAPIKeyResponse `json:"api_key"`
	}
	decodeBody(t, createResp, &created)
	if created.APIKey.Token == "" {
		t.Fatalf("expected one-time token in create response")
	}
	if created.APIKey.Key.TokenPreview == "" {
		t.Fatalf("expected token preview in create response")
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/account/api-keys", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200 listing api keys, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		APIKeys []accountAPIKeyResponse `json:"api_keys"`
	}
	decodeBody(t, listResp, &listed)
	if len(listed.APIKeys) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(listed.APIKeys))
	}

	keyID := created.APIKey.Key.ID
	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account/api-keys/"+keyID+":update",
		bytes.NewBufferString(`{"name":"Studio automation v2","scope":"admin"}`),
	)
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 updating api key, got %d: %s", updateResp.Code, updateResp.Body.String())
	}
	var updated struct {
		APIKey accountAPIKeyResponse `json:"api_key"`
	}
	decodeBody(t, updateResp, &updated)
	if updated.APIKey.Scope != "admin" {
		t.Fatalf("expected admin scope after update, got %q", updated.APIKey.Scope)
	}

	toggleResp := httptest.NewRecorder()
	toggleReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account/api-keys/"+keyID+":toggle",
		bytes.NewBufferString(`{"is_active":false}`),
	)
	router.ServeHTTP(toggleResp, toggleReq)
	if toggleResp.Code != http.StatusOK {
		t.Fatalf("expected 200 toggling api key, got %d: %s", toggleResp.Code, toggleResp.Body.String())
	}
	var toggled struct {
		APIKey accountAPIKeyResponse `json:"api_key"`
	}
	decodeBody(t, toggleResp, &toggled)
	if toggled.APIKey.IsActive {
		t.Fatalf("expected api key to be inactive")
	}

	deleteResp := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account/api-keys/"+keyID+":delete",
		nil,
	)
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting api key, got %d: %s", deleteResp.Code, deleteResp.Body.String())
	}
}
