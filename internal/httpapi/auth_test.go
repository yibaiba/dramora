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
