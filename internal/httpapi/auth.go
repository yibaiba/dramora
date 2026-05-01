package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type authRequest struct {
	Email           string `json:"email"`
	DisplayName     string `json:"display_name"`
	Password        string `json:"password"`
	InvitationToken string `json:"invitation_token,omitempty"`
}

type authSessionResponse struct {
	Token            string       `json:"token"`
	User             userResponse `json:"user"`
	OrganizationID   string       `json:"organization_id"`
	Role             string       `json:"role"`
	ExpiresAt        time.Time    `json:"expires_at"`
	RefreshToken     string       `json:"refresh_token,omitempty"`
	RefreshExpiresAt *time.Time   `json:"refresh_expires_at,omitempty"`
	CurrentSessionID string       `json:"current_session_id,omitempty"`
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func userDTO(user domain.User) userResponse {
	return userResponse{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
}

func authSessionDTO(session service.AuthSession) authSessionResponse {
	resp := authSessionResponse{
		Token:            session.Token,
		User:             userDTO(session.User),
		OrganizationID:   session.OrganizationID,
		Role:             session.Role,
		ExpiresAt:        session.ExpiresAt.UTC(),
		RefreshToken:     session.RefreshToken,
		CurrentSessionID: session.RefreshTokenID,
	}
	if !session.RefreshExpiresAt.IsZero() {
		t := session.RefreshExpiresAt.UTC()
		resp.RefreshExpiresAt = &t
	}
	return resp
}

func (a *api) register(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	session, err := a.authService.Register(r.Context(), service.RegisterInput{
		Email:           request.Email,
		DisplayName:     request.DisplayName,
		Password:        request.Password,
		InvitationToken: request.InvitationToken,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	session, err := a.authService.Login(r.Context(), service.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func (a *api) currentSession(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	session, err := a.authService.CurrentSession(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *api) refreshSession(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	session, err := a.authService.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func (a *api) logoutSession(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var request refreshRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	// 主动忽略 logout 错误（除存储级故障外），避免泄露 token 是否存在。
	if err := a.authService.Logout(r.Context(), request.RefreshToken); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired credentials")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeServiceError(w, err)
	}
}

type invitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type invitationResponse struct {
	ID              string     `json:"id"`
	OrganizationID  string     `json:"organization_id"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	Token           string     `json:"token"`
	Status          string     `json:"status"`
	InvitedByUserID string     `json:"invited_by_user_id,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	AcceptedByUser  string     `json:"accepted_by_user_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func invitationDTO(inv domain.OrganizationInvitation) invitationResponse {
	return invitationResponse{
		ID:              inv.ID,
		OrganizationID:  inv.OrganizationID,
		Email:           inv.Email,
		Role:            inv.Role,
		Token:           inv.Token,
		Status:          inv.Status,
		InvitedByUserID: inv.InvitedByUserID,
		ExpiresAt:       inv.ExpiresAt.UTC(),
		AcceptedAt:      inv.AcceptedAt,
		AcceptedByUser:  inv.AcceptedByUserID,
		CreatedAt:       inv.CreatedAt.UTC(),
	}
}

func (a *api) createInvitation(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request invitationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	inv, err := a.authService.CreateInvitation(r.Context(), service.CreateInvitationInput{
		Email: request.Email,
		Role:  request.Role,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]invitationResponse{"invitation": invitationDTO(inv)})
}

func (a *api) listInvitations(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	items, err := a.authService.ListInvitations(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]invitationResponse, 0, len(items))
	for _, inv := range items {
		out = append(out, invitationDTO(inv))
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": out})
}

func (a *api) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	invitationID := chi.URLParam(r, "invitationId")
	if err := a.authService.RevokeInvitation(r.Context(), invitationID); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) resendInvitation(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	invitationID := chi.URLParam(r, "invitationId")
	inv, err := a.authService.ResendInvitation(r.Context(), invitationID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]invitationResponse{"invitation": invitationDTO(inv)})
}

type invitationAuditEventResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	InvitationID   string    `json:"invitation_id"`
	Action         string    `json:"action"`
	ActorUserID    string    `json:"actor_user_id,omitempty"`
	ActorEmail     string    `json:"actor_email,omitempty"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	Note           string    `json:"note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func invitationAuditDTO(ev domain.InvitationAuditEvent) invitationAuditEventResponse {
	return invitationAuditEventResponse{
		ID:             ev.ID,
		OrganizationID: ev.OrganizationID,
		InvitationID:   ev.InvitationID,
		Action:         ev.Action,
		ActorUserID:    ev.ActorUserID,
		ActorEmail:     ev.ActorEmail,
		Email:          ev.Email,
		Role:           ev.Role,
		Note:           ev.Note,
		CreatedAt:      ev.CreatedAt.UTC(),
	}
}

// parseInvitationAuditFilter parses common filter query params shared by list/export endpoints.
// Returns true on success; on failure it writes an error response and returns false.
func parseInvitationAuditFilter(w http.ResponseWriter, r *http.Request, defaultLimit, maxLimit int) (repo.InvitationAuditFilter, bool) {
	q := r.URL.Query()
	filter := repo.InvitationAuditFilter{
		Email: strings.TrimSpace(q.Get("email")),
	}
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			if parsed > maxLimit {
				parsed = maxLimit
			}
			filter.Limit = parsed
		}
	}
	if filter.Limit == 0 {
		filter.Limit = defaultLimit
	}
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			filter.Offset = parsed
		}
	}
	if raw := strings.TrimSpace(q.Get("action")); raw != "" {
		actions := make([]string, 0, 4)
		seen := map[string]struct{}{}
		for _, part := range strings.Split(raw, ",") {
			a := strings.TrimSpace(part)
			if a == "" {
				continue
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			actions = append(actions, a)
		}
		filter.Actions = actions
	}
	if raw := strings.TrimSpace(q.Get("since")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.Since = &t
		} else {
			writeError(w, http.StatusBadRequest, "invalid_since", "since must be RFC3339")
			return filter, false
		}
	}
	if raw := strings.TrimSpace(q.Get("until")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.Until = &t
		} else {
			writeError(w, http.StatusBadRequest, "invalid_until", "until must be RFC3339")
			return filter, false
		}
	}
	return filter, true
}

func (a *api) listInvitationAudit(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	filter, ok := parseInvitationAuditFilter(w, r, 50, 200)
	if !ok {
		return
	}
	page, err := a.authService.ListInvitationAuditEvents(r.Context(), filter)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]invitationAuditEventResponse, 0, len(page.Events))
	for _, ev := range page.Events {
		out = append(out, invitationAuditDTO(ev))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events":   out,
		"has_more": page.HasMore,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	})
}

// exportInvitationAudit streams the filtered audit log as JSON or CSV.
// Pagination is bypassed by raising the per-request cap to 5000; offset is ignored.
func (a *api) exportInvitationAudit(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		writeError(w, http.StatusBadRequest, "invalid_format", "format must be csv or json")
		return
	}
	filter, ok := parseInvitationAuditFilter(w, r, 5000, 5000)
	if !ok {
		return
	}
	filter.Offset = 0
	page, err := a.authService.ListInvitationAuditEvents(r.Context(), filter)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	timestamp := time.Now().UTC().Format("20060102-150405")
	out := make([]invitationAuditEventResponse, 0, len(page.Events))
	for _, ev := range page.Events {
		out = append(out, invitationAuditDTO(ev))
	}
	filterSummary := map[string]any{
		"actions": filter.Actions,
		"email":   filter.Email,
	}
	if filter.Since != nil {
		filterSummary["since"] = filter.Since.UTC().Format(time.RFC3339)
	}
	if filter.Until != nil {
		filterSummary["until"] = filter.Until.UTC().Format(time.RFC3339)
	}
	if format == "json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="invitation-audit-`+timestamp+`.json"`)
		w.Header().Set("X-Has-More", strconv.FormatBool(page.HasMore))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events":         out,
			"has_more":       page.HasMore,
			"exported_at":    time.Now().UTC().Format(time.RFC3339),
			"event_count":    len(out),
			"filter_summary": filterSummary,
		})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="invitation-audit-`+timestamp+`.csv"`)
	w.Header().Set("X-Has-More", strconv.FormatBool(page.HasMore))
	w.WriteHeader(http.StatusOK)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"id", "organization_id", "invitation_id", "action", "email", "role", "actor_email", "actor_user_id", "note", "created_at"})
	for _, ev := range out {
		_ = cw.Write([]string{
			ev.ID,
			ev.OrganizationID,
			ev.InvitationID,
			ev.Action,
			ev.Email,
			ev.Role,
			ev.ActorEmail,
			ev.ActorUserID,
			ev.Note,
			ev.CreatedAt.Format(time.RFC3339),
		})
	}
}

type sessionResponse struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Role           string     `json:"role"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	ReplacedByID   string     `json:"replaced_by_id,omitempty"`
}

func sessionDTO(info service.SessionInfo) sessionResponse {
	resp := sessionResponse{
		ID:             info.ID,
		OrganizationID: info.OrganizationID,
		Role:           info.Role,
		CreatedAt:      info.CreatedAt.UTC(),
		ExpiresAt:      info.ExpiresAt.UTC(),
	}
	if info.RevokedAt != nil {
		t := info.RevokedAt.UTC()
		resp.RevokedAt = &t
	}
	if info.ReplacedByID != nil {
		resp.ReplacedByID = *info.ReplacedByID
	}
	return resp
}

func (a *api) listSessions(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	items, err := a.authService.ListSessions(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]sessionResponse, 0, len(items))
	for _, info := range items {
		out = append(out, sessionDTO(info))
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (a *api) revokeSession(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	sessionID := chi.URLParam(r, "sessionId")
	if err := a.authService.RevokeSession(r.Context(), sessionID); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
