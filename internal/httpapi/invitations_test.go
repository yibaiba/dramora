package httpapi

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestInvitationAuditLogCapturesLifecycle(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// 1. Create initial invitation -> expect 1 `created` audit event.
	createBody := bytes.NewBufferString(`{"email":"audit-me@example.com","role":"editor"}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", createBody)
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create invitation: expected 201, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		Invitation invitationResponse `json:"invitation"`
	}
	decodeBody(t, createResp, &created)

	// 2. Resend it -> emits `revoked` (old) + `created` (new) audit events.
	resendResp := httptest.NewRecorder()
	resendReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/"+created.Invitation.ID+":resend", nil)
	router.ServeHTTP(resendResp, resendReq)
	if resendResp.Code != http.StatusCreated {
		t.Fatalf("resend invitation: expected 201, got %d: %s", resendResp.Code, resendResp.Body.String())
	}
	var resent struct {
		Invitation invitationResponse `json:"invitation"`
	}
	decodeBody(t, resendResp, &resent)

	// 3. Register the resent invitation -> emits `accepted` audit event.
	regBody, _ := json.Marshal(map[string]string{
		"email":            "audit-me@example.com",
		"display_name":     "Audit Me",
		"password":         "strongpass",
		"invitation_token": resent.Invitation.Token,
	})
	regResp := httptest.NewRecorder()
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	router.ServeHTTP(regResp, regReq)
	if regResp.Code != http.StatusCreated {
		t.Fatalf("register with resent token: expected 201, got %d: %s", regResp.Code, regResp.Body.String())
	}

	// 4. List audit events.
	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list audit: expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Events []invitationAuditEventResponse `json:"events"`
	}
	decodeBody(t, listResp, &listed)

	// Tally per-action counts and ensure invitation_id linkage is correct.
	counts := map[string]int{}
	var sawAcceptedForResent bool
	for _, ev := range listed.Events {
		counts[ev.Action]++
		if ev.Action == "accepted" && ev.InvitationID == resent.Invitation.ID {
			sawAcceptedForResent = true
		}
		if ev.Email != "audit-me@example.com" {
			t.Fatalf("expected audit event email snapshot to match invitee, got %q", ev.Email)
		}
	}
	if counts["created"] < 2 {
		t.Fatalf("expected ≥2 created events (initial + resend), got %d", counts["created"])
	}
	if counts["revoked"] < 1 {
		t.Fatalf("expected ≥1 revoked event from resend, got %d", counts["revoked"])
	}
	if counts["accepted"] != 1 {
		t.Fatalf("expected exactly 1 accepted event, got %d", counts["accepted"])
	}
	if !sawAcceptedForResent {
		t.Fatalf("expected accepted event to reference resent invitation %q, events=%+v", resent.Invitation.ID, listed.Events)
	}

	// 5. Filter by action=revoked -> only revoked events.
	revResp := httptest.NewRecorder()
	revReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit?action=revoked", nil)
	router.ServeHTTP(revResp, revReq)
	if revResp.Code != http.StatusOK {
		t.Fatalf("filter by action: expected 200, got %d: %s", revResp.Code, revResp.Body.String())
	}
	var filtered struct {
		Events  []invitationAuditEventResponse `json:"events"`
		HasMore bool                           `json:"has_more"`
		Limit   int                            `json:"limit"`
	}
	decodeBody(t, revResp, &filtered)
	if len(filtered.Events) == 0 {
		t.Fatalf("expected at least one revoked event after filter, got 0")
	}
	for _, ev := range filtered.Events {
		if ev.Action != "revoked" {
			t.Fatalf("expected only revoked events, got %q", ev.Action)
		}
	}
	if filtered.Limit != 50 {
		t.Fatalf("expected default limit 50, got %d", filtered.Limit)
	}

	// 6. Pagination: limit=1 should return 1 item with has_more=true.
	pageResp := httptest.NewRecorder()
	pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit?limit=1", nil)
	router.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("paginate: expected 200, got %d: %s", pageResp.Code, pageResp.Body.String())
	}
	var paged struct {
		Events  []invitationAuditEventResponse `json:"events"`
		HasMore bool                           `json:"has_more"`
	}
	decodeBody(t, pageResp, &paged)
	if len(paged.Events) != 1 {
		t.Fatalf("expected 1 event with limit=1, got %d", len(paged.Events))
	}
	if !paged.HasMore {
		t.Fatalf("expected has_more=true with limit=1 and ≥2 audit events")
	}

	// 7. Email filter (case-insensitive substring) hits all events for this invitee.
	emailResp := httptest.NewRecorder()
	emailReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit?email=AUDIT-ME", nil)
	router.ServeHTTP(emailResp, emailReq)
	if emailResp.Code != http.StatusOK {
		t.Fatalf("email filter: expected 200, got %d: %s", emailResp.Code, emailResp.Body.String())
	}
	var byEmail struct {
		Events []invitationAuditEventResponse `json:"events"`
	}
	decodeBody(t, emailResp, &byEmail)
	if len(byEmail.Events) < 4 {
		t.Fatalf("expected ≥4 events for email filter (created+revoked+created+accepted), got %d", len(byEmail.Events))
	}

	// 8. Email filter that doesn't match -> empty.
	missResp := httptest.NewRecorder()
	missReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit?email=nobody-here", nil)
	router.ServeHTTP(missResp, missReq)
	if missResp.Code != http.StatusOK {
		t.Fatalf("email miss: expected 200, got %d", missResp.Code)
	}
	var miss struct {
		Events []invitationAuditEventResponse `json:"events"`
	}
	decodeBody(t, missResp, &miss)
	if len(miss.Events) != 0 {
		t.Fatalf("expected 0 events for non-matching email filter, got %d", len(miss.Events))
	}

	// 9. Invalid since -> 400.
	badResp := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit?since=not-a-date", nil)
	router.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid since: expected 400, got %d", badResp.Code)
	}
}

func TestInvitationAuditExport(t *testing.T) {
	t.Parallel()

	router := testRouter()

	// Seed a couple of invitations so the export has rows.
	for _, email := range []string{"export-a@example.com", "export-b@example.com"} {
		body := bytes.NewBufferString(`{"email":"` + email + `","role":"editor"}`)
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", body)
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("seed invitation %s: expected 201, got %d: %s", email, resp.Code, resp.Body.String())
		}
	}

	// CSV export (default).
	csvResp := httptest.NewRecorder()
	csvReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit/export", nil)
	router.ServeHTTP(csvResp, csvReq)
	if csvResp.Code != http.StatusOK {
		t.Fatalf("csv export: expected 200, got %d: %s", csvResp.Code, csvResp.Body.String())
	}
	if ct := csvResp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("csv export: expected text/csv content-type, got %q", ct)
	}
	if cd := csvResp.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment;") || !strings.Contains(cd, ".csv") {
		t.Fatalf("csv export: expected attachment .csv disposition, got %q", cd)
	}
	rows, err := csv.NewReader(csvResp.Body).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) < 3 {
		t.Fatalf("expected header + ≥2 data rows, got %d rows", len(rows))
	}
	if rows[0][0] != "id" || rows[0][3] != "action" || rows[0][9] != "created_at" {
		t.Fatalf("unexpected csv header: %+v", rows[0])
	}

	// JSON export with action filter.
	jsonResp := httptest.NewRecorder()
	jsonReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit/export?format=json&action=created", nil)
	router.ServeHTTP(jsonResp, jsonReq)
	if jsonResp.Code != http.StatusOK {
		t.Fatalf("json export: expected 200, got %d: %s", jsonResp.Code, jsonResp.Body.String())
	}
	if ct := jsonResp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("json export: expected application/json, got %q", ct)
	}
	if cd := jsonResp.Header().Get("Content-Disposition"); !strings.Contains(cd, ".json") {
		t.Fatalf("json export: expected .json disposition, got %q", cd)
	}
	var payload struct {
		Events  []invitationAuditEventResponse `json:"events"`
		HasMore bool                           `json:"has_more"`
	}
	if err := json.Unmarshal(jsonResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json export: %v", err)
	}
	if len(payload.Events) < 2 {
		t.Fatalf("expected ≥2 created events in json export, got %d", len(payload.Events))
	}
	for _, ev := range payload.Events {
		if ev.Action != "created" {
			t.Fatalf("json export action filter: expected only created, got %q", ev.Action)
		}
	}

	// Unknown format -> 400.
	badResp := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/invitations/audit/export?format=xml", nil)
	router.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid format: expected 400, got %d", badResp.Code)
	}
}
