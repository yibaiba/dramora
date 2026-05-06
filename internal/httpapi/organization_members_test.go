package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrganizationMembersLifecycle(t *testing.T) {
	t.Parallel()

	router := testRouter()
	memberSession := registerInvitedUser(t, router, "member@example.com", "editor")

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/members", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200 listing members, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Members []organizationMemberResponse `json:"members"`
	}
	decodeBody(t, listResp, &listed)
	if len(listed.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(listed.Members))
	}
	var found bool
	for _, member := range listed.Members {
		if member.UserID == memberSession.User.ID {
			found = true
			if member.Role != "editor" {
				t.Fatalf("expected invited member role editor, got %q", member.Role)
			}
			if member.LastActivityAt == "" {
				t.Fatalf("expected last_activity_at to be populated")
			}
		}
	}
	if !found {
		t.Fatalf("expected invited member in response, got %+v", listed.Members)
	}

	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations/members/"+memberSession.User.ID+"/role",
		bytes.NewBufferString(`{"role":"admin"}`),
	)
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 updating role, got %d: %s", updateResp.Code, updateResp.Body.String())
	}
	var updated struct {
		Member organizationMemberResponse `json:"member"`
	}
	decodeBody(t, updateResp, &updated)
	if updated.Member.Role != "admin" {
		t.Fatalf("expected admin role after update, got %q", updated.Member.Role)
	}

	removeResp := httptest.NewRecorder()
	removeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations/members/"+memberSession.User.ID+":remove",
		nil,
	)
	router.ServeHTTP(removeResp, removeReq)
	if removeResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 removing member, got %d: %s", removeResp.Code, removeResp.Body.String())
	}

	verifyResp := httptest.NewRecorder()
	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/members", nil)
	router.ServeHTTP(verifyResp, verifyReq)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("expected 200 re-listing members, got %d: %s", verifyResp.Code, verifyResp.Body.String())
	}
	var verified struct {
		Members []organizationMemberResponse `json:"members"`
	}
	decodeBody(t, verifyResp, &verified)
	if len(verified.Members) != 1 {
		t.Fatalf("expected 1 member after removal, got %d", len(verified.Members))
	}
}

func TestOrganizationMembersRejectRemovingLastOwner(t *testing.T) {
	t.Parallel()

	router := testRouter()
	session := currentSession(t, router)

	removeResp := httptest.NewRecorder()
	removeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations/members/"+session.User.ID+":remove",
		nil,
	)
	router.ServeHTTP(removeResp, removeReq)
	if removeResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 removing last owner, got %d: %s", removeResp.Code, removeResp.Body.String())
	}

	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations/members/"+session.User.ID+"/role",
		bytes.NewBufferString(`{"role":"admin"}`),
	)
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 demoting last owner, got %d: %s", updateResp.Code, updateResp.Body.String())
	}
}

func registerInvitedUser(t *testing.T, router http.Handler, email, role string) authSessionResponse {
	t.Helper()

	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations/invitations",
		bytes.NewBufferString(`{"email":"`+email+`","role":"`+role+`"}`),
	)
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating invitation, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		Invitation invitationResponse `json:"invitation"`
	}
	decodeBody(t, createResp, &created)

	payload, err := json.Marshal(map[string]string{
		"email":            email,
		"display_name":     "Invited Member",
		"password":         "strongpass",
		"invitation_token": created.Invitation.Token,
	})
	if err != nil {
		t.Fatalf("marshal invited user: %v", err)
	}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 registering invited user, got %d: %s", resp.Code, resp.Body.String())
	}
	var session struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, resp, &session)
	return session.Session
}

func currentSession(t *testing.T, router http.Handler) authSessionResponse {
	t.Helper()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 current session, got %d: %s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, resp, &payload)
	return payload.Session
}
