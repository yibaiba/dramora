package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAutoProvisionsOrganization(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// Register two fresh users (in addition to the default test director created
	// by newAuthenticatedTestRouter). They should each land in their own
	// freshly provisioned organization, NOT the legacy default org.
	first := registerUser(t, router, "alice@example.com", "Alice")
	second := registerUser(t, router, "bob@example.com", "Bob")

	if first.OrganizationID == "" || second.OrganizationID == "" {
		t.Fatalf("expected non-empty org ids, got %q / %q", first.OrganizationID, second.OrganizationID)
	}
	if first.OrganizationID == second.OrganizationID {
		t.Fatalf("expected unique organizations per registration, both got %q", first.OrganizationID)
	}
	if first.Role != "owner" || second.Role != "owner" {
		t.Fatalf("expected owner role on auto-provisioned orgs, got %q / %q", first.Role, second.Role)
	}
}

func TestInvitationFlow(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// 1. Default test director (owner) creates an invitation.
	createBody := bytes.NewBufferString(`{"email":"invitee@example.com","role":"editor"}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", createBody)
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating invitation, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		Invitation invitationResponse `json:"invitation"`
	}
	decodeBody(t, createResp, &created)
	if created.Invitation.Token == "" {
		t.Fatalf("expected invitation token in response")
	}

	directorOrgID := created.Invitation.OrganizationID

	// 2. Register a brand-new user with that token; they should join the director's org.
	body := map[string]string{
		"email":            "invitee@example.com",
		"display_name":     "Invitee",
		"password":         "strongpass",
		"invitation_token": created.Invitation.Token,
	}
	raw, _ := json.Marshal(body)
	regResp := httptest.NewRecorder()
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(raw))
	router.ServeHTTP(regResp, regReq)
	if regResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 registering invitee, got %d: %s", regResp.Code, regResp.Body.String())
	}
	var session struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, regResp, &session)
	if session.Session.OrganizationID != directorOrgID {
		t.Fatalf("expected invitee org %q, got %q", directorOrgID, session.Session.OrganizationID)
	}
	if session.Session.Role != "editor" {
		t.Fatalf("expected editor role from invitation, got %q", session.Session.Role)
	}

	// 3. Reusing the now-accepted token should fail.
	replay := bytes.NewBuffer(raw)
	replayResp := httptest.NewRecorder()
	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", replay)
	router.ServeHTTP(replayResp, replayReq)
	if replayResp.Code == http.StatusCreated {
		t.Fatalf("expected accepted invitation to be rejected on replay, got 201")
	}
}

func registerUser(t *testing.T, router http.Handler, email, name string) authSessionResponse {
	t.Helper()
	body := map[string]string{"email": email, "display_name": name, "password": "strongpass"}
	raw, _ := json.Marshal(body)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(raw))
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", email, resp.Code, resp.Body.String())
	}
	var session struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, resp, &session)
	return session.Session
}
