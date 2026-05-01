package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthRegisterLoginAndMe(t *testing.T) {
	t.Parallel()

	router := testRouter()

	registerResp := httptest.NewRecorder()
	registerReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"director@example.com","display_name":"Director","password":"strongpass"}`),
	)
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d: %s", registerResp.Code, registerResp.Body.String())
	}

	var registerPayload struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, registerResp, &registerPayload)
	if registerPayload.Session.User.Email != "director@example.com" {
		t.Fatalf("expected registered email, got %q", registerPayload.Session.User.Email)
	}
	if registerPayload.Session.Token == "" {
		t.Fatalf("expected register token")
	}

	loginResp := httptest.NewRecorder()
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		body(`{"email":"director@example.com","password":"strongpass"}`),
	)
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var loginPayload struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, loginResp, &loginPayload)
	if loginPayload.Session.OrganizationID == "" {
		t.Fatalf("expected organization id")
	}

	meResp := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginPayload.Session.Token)
	router.ServeHTTP(meResp, meReq)
	if meResp.Code != http.StatusOK {
		t.Fatalf("expected me 200, got %d: %s", meResp.Code, meResp.Body.String())
	}
}

func TestAuthLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	router := testRouter()

	registerResp := httptest.NewRecorder()
	registerReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"director@example.com","display_name":"Director","password":"strongpass"}`),
	)
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d: %s", registerResp.Code, registerResp.Body.String())
	}

	loginResp := httptest.NewRecorder()
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		body(`{"email":"director@example.com","password":"wrongpass"}`),
	)
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected login 401, got %d: %s", loginResp.Code, loginResp.Body.String())
	}
}

func TestAuthRefreshRotatesAndLogoutInvalidates(t *testing.T) {
	t.Parallel()

	router := testRouter()

	registerResp := httptest.NewRecorder()
	registerReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"director@example.com","display_name":"Director","password":"strongpass"}`),
	)
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d: %s", registerResp.Code, registerResp.Body.String())
	}

	var registered struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, registerResp, &registered)
	if registered.Session.RefreshToken == "" {
		t.Fatalf("expected refresh_token on register response")
	}
	if registered.Session.RefreshExpiresAt == nil {
		t.Fatalf("expected refresh_expires_at on register response")
	}

	originalRefresh := registered.Session.RefreshToken

	refreshResp := httptest.NewRecorder()
	refreshReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		body(`{"refresh_token":"`+originalRefresh+`"}`),
	)
	router.ServeHTTP(refreshResp, refreshReq)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("expected refresh 200, got %d: %s", refreshResp.Code, refreshResp.Body.String())
	}

	var refreshed struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, refreshResp, &refreshed)
	if refreshed.Session.RefreshToken == "" || refreshed.Session.RefreshToken == originalRefresh {
		t.Fatalf("expected rotated refresh token, got %q", refreshed.Session.RefreshToken)
	}
	if refreshed.Session.Token == "" {
		t.Fatalf("expected new access token")
	}

	// 旧 token 已被吊销，再次刷新应失败。
	reuseResp := httptest.NewRecorder()
	reuseReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		body(`{"refresh_token":"`+originalRefresh+`"}`),
	)
	router.ServeHTTP(reuseResp, reuseReq)
	if reuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected reused refresh 401, got %d: %s", reuseResp.Code, reuseResp.Body.String())
	}

	// logout 后新 token 也失效。
	logoutResp := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		body(`{"refresh_token":"`+refreshed.Session.RefreshToken+`"}`),
	)
	router.ServeHTTP(logoutResp, logoutReq)
	if logoutResp.Code != http.StatusNoContent {
		t.Fatalf("expected logout 204, got %d: %s", logoutResp.Code, logoutResp.Body.String())
	}

	postLogoutResp := httptest.NewRecorder()
	postLogoutReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		body(`{"refresh_token":"`+refreshed.Session.RefreshToken+`"}`),
	)
	router.ServeHTTP(postLogoutResp, postLogoutReq)
	if postLogoutResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected post-logout refresh 401, got %d: %s", postLogoutResp.Code, postLogoutResp.Body.String())
	}
}

func TestAuthListAndRevokeOwnSessions(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// Register user A — produces session 1.
	regA := httptest.NewRecorder()
	router.ServeHTTP(regA, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"a@example.com","display_name":"User A","password":"strongpass"}`),
	))
	if regA.Code != http.StatusCreated {
		t.Fatalf("register A: %d %s", regA.Code, regA.Body.String())
	}
	var registered struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, regA, &registered)
	if registered.Session.CurrentSessionID == "" {
		t.Fatalf("expected current_session_id on register response")
	}
	tokenA := registered.Session.Token
	currentID := registered.Session.CurrentSessionID

	// Login A again — produces session 2.
	loginA := httptest.NewRecorder()
	router.ServeHTTP(loginA, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		body(`{"email":"a@example.com","password":"strongpass"}`),
	))
	if loginA.Code != http.StatusOK {
		t.Fatalf("login A: %d %s", loginA.Code, loginA.Body.String())
	}
	var second struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, loginA, &second)
	secondID := second.Session.CurrentSessionID
	secondRefresh := second.Session.RefreshToken
	if secondID == "" || secondID == currentID {
		t.Fatalf("expected distinct second session id, got %q vs %q", secondID, currentID)
	}

	// List A's sessions.
	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenA)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list sessions: %d %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Sessions []sessionResponse `json:"sessions"`
	}
	decodeBody(t, listResp, &listed)
	if len(listed.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(listed.Sessions))
	}

	// Revoke the second session.
	revokeResp := httptest.NewRecorder()
	revokeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/sessions/"+secondID+":revoke",
		nil,
	)
	revokeReq.Header.Set("Authorization", "Bearer "+tokenA)
	router.ServeHTTP(revokeResp, revokeReq)
	if revokeResp.Code != http.StatusNoContent {
		t.Fatalf("revoke session: %d %s", revokeResp.Code, revokeResp.Body.String())
	}

	// Refreshing with the revoked token should fail.
	refreshResp := httptest.NewRecorder()
	router.ServeHTTP(refreshResp, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		body(`{"refresh_token":"`+secondRefresh+`"}`),
	))
	if refreshResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked refresh 401, got %d: %s", refreshResp.Code, refreshResp.Body.String())
	}

	// Register user B and confirm B cannot revoke A's sessions (404 — no leak).
	regB := httptest.NewRecorder()
	router.ServeHTTP(regB, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"b@example.com","display_name":"User B","password":"strongpass"}`),
	))
	if regB.Code != http.StatusCreated {
		t.Fatalf("register B: %d %s", regB.Code, regB.Body.String())
	}
	var bSession struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, regB, &bSession)

	crossResp := httptest.NewRecorder()
	crossReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/sessions/"+currentID+":revoke",
		nil,
	)
	crossReq.Header.Set("Authorization", "Bearer "+bSession.Session.Token)
	router.ServeHTTP(crossResp, crossReq)
	if crossResp.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user revoke 404, got %d: %s", crossResp.Code, crossResp.Body.String())
	}

	// B sees only own sessions.
	listB := httptest.NewRecorder()
	listBReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	listBReq.Header.Set("Authorization", "Bearer "+bSession.Session.Token)
	router.ServeHTTP(listB, listBReq)
	if listB.Code != http.StatusOK {
		t.Fatalf("list B sessions: %d %s", listB.Code, listB.Body.String())
	}
	var listedB struct {
		Sessions []sessionResponse `json:"sessions"`
	}
	decodeBody(t, listB, &listedB)
	if len(listedB.Sessions) != 1 {
		t.Fatalf("expected exactly 1 session for B, got %d", len(listedB.Sessions))
	}
}
