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

func TestRevokeInvitationRejectsReuse(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// 1. Default test director (owner) creates an invitation.
	createBody := bytes.NewBufferString(`{"email":"revoke-me@example.com","role":"editor"}`)
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
	if created.Invitation.ID == "" || created.Invitation.Token == "" {
		t.Fatalf("expected invitation id and token, got %+v", created.Invitation)
	}

	// 2. Owner revokes the invitation -> 204.
	revokeResp := httptest.NewRecorder()
	revokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/"+created.Invitation.ID+":revoke", nil)
	router.ServeHTTP(revokeResp, revokeReq)
	if revokeResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 revoking invitation, got %d: %s", revokeResp.Code, revokeResp.Body.String())
	}

	// 3. Second revoke (already revoked) -> 404 (status no longer pending).
	secondResp := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/"+created.Invitation.ID+":revoke", nil)
	router.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 revoking already-revoked invitation, got %d: %s", secondResp.Code, secondResp.Body.String())
	}

	// 4. Registering with the revoked token must fail.
	regBody := map[string]string{
		"email":            "revoke-me@example.com",
		"display_name":     "Revoked",
		"password":         "strongpass",
		"invitation_token": created.Invitation.Token,
	}
	raw, _ := json.Marshal(regBody)
	regResp := httptest.NewRecorder()
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(raw))
	router.ServeHTTP(regResp, regReq)
	if regResp.Code == http.StatusCreated {
		t.Fatalf("expected revoked invitation token to be rejected, got 201: %s", regResp.Body.String())
	}

	// 5. List should now show status=revoked for the invitation.
	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200 listing invitations, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var list struct {
		Invitations []invitationResponse `json:"invitations"`
	}
	decodeBody(t, listResp, &list)
	var found *invitationResponse
	for i := range list.Invitations {
		if list.Invitations[i].ID == created.Invitation.ID {
			found = &list.Invitations[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected revoked invitation %q in list, got %+v", created.Invitation.ID, list.Invitations)
	}
	if found.Status != "revoked" {
		t.Fatalf("expected revoked status on listed invitation, got %q", found.Status)
	}
}

func TestResendInvitationRevokesOldAndIssuesNewToken(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// 1. Create the original invitation.
	createBody := bytes.NewBufferString(`{"email":"resend-me@example.com","role":"editor"}`)
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
	if created.Invitation.ID == "" || created.Invitation.Token == "" {
		t.Fatalf("expected invitation id+token, got %+v", created.Invitation)
	}

	// 2. Resend -> 201 with a brand new invitation (new id + new token).
	resendResp := httptest.NewRecorder()
	resendReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/"+created.Invitation.ID+":resend", nil)
	router.ServeHTTP(resendResp, resendReq)
	if resendResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 resending invitation, got %d: %s", resendResp.Code, resendResp.Body.String())
	}
	var resent struct {
		Invitation invitationResponse `json:"invitation"`
	}
	decodeBody(t, resendResp, &resent)
	if resent.Invitation.ID == "" || resent.Invitation.Token == "" {
		t.Fatalf("expected resent invitation id+token, got %+v", resent.Invitation)
	}
	if resent.Invitation.ID == created.Invitation.ID {
		t.Fatalf("expected new invitation id, got same %q", resent.Invitation.ID)
	}
	if resent.Invitation.Token == created.Invitation.Token {
		t.Fatalf("expected new invitation token, got same %q", resent.Invitation.Token)
	}
	if resent.Invitation.Email != "resend-me@example.com" || resent.Invitation.Role != "editor" {
		t.Fatalf("expected email/role preserved, got %+v", resent.Invitation)
	}
	if resent.Invitation.Status != "pending" {
		t.Fatalf("expected new invitation pending, got %q", resent.Invitation.Status)
	}

	// 3. Resending the original (now revoked) id again -> 404.
	secondResp := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/"+created.Invitation.ID+":resend", nil)
	router.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 resending revoked invitation, got %d: %s", secondResp.Code, secondResp.Body.String())
	}

	// 4. Original token should no longer be usable for registration; new token must work.
	oldRegBody, _ := json.Marshal(map[string]string{
		"email":            "resend-me@example.com",
		"display_name":     "Old Token",
		"password":         "strongpass",
		"invitation_token": created.Invitation.Token,
	})
	oldRegResp := httptest.NewRecorder()
	oldRegReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(oldRegBody))
	router.ServeHTTP(oldRegResp, oldRegReq)
	if oldRegResp.Code == http.StatusCreated {
		t.Fatalf("expected old token to be rejected after resend, got 201: %s", oldRegResp.Body.String())
	}

	newRegBody, _ := json.Marshal(map[string]string{
		"email":            "resend-me@example.com",
		"display_name":     "New Token",
		"password":         "strongpass",
		"invitation_token": resent.Invitation.Token,
	})
	newRegResp := httptest.NewRecorder()
	newRegReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(newRegBody))
	router.ServeHTTP(newRegResp, newRegReq)
	if newRegResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 registering with new resent token, got %d: %s", newRegResp.Code, newRegResp.Body.String())
	}
}
