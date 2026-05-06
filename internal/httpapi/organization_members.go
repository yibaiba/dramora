package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type organizationMemberResponse struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	DisplayName    string `json:"display_name"`
	Role           string `json:"role"`
	JoinedAt       string `json:"joined_at"`
	LastActivityAt string `json:"last_activity_at"`
}

type updateOrganizationMemberRoleRequest struct {
	Role string `json:"role"`
}

func organizationMemberDTO(member service.OrganizationMemberInfo) organizationMemberResponse {
	return organizationMemberResponse{
		UserID:         member.UserID,
		OrganizationID: member.OrganizationID,
		Email:          member.Email,
		DisplayName:    member.DisplayName,
		Role:           member.Role,
		JoinedAt:       member.JoinedAt.UTC().Format(time.RFC3339),
		LastActivityAt: member.LastActivityAt.UTC().Format(time.RFC3339),
	}
}

func (a *api) listOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	items, err := a.authService.ListOrganizationMembers(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]organizationMemberResponse, 0, len(items))
	for _, member := range items {
		out = append(out, organizationMemberDTO(member))
	}
	writeJSON(w, http.StatusOK, Envelope{"members": out})
}

func (a *api) updateOrganizationMemberRole(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request updateOrganizationMemberRoleRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}
	member, err := a.authService.UpdateOrganizationMemberRole(r.Context(), chi.URLParam(r, "userId"), request.Role)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"member": organizationMemberDTO(member)})
}

func (a *api) removeOrganizationMember(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	if err := a.authService.RemoveOrganizationMember(r.Context(), chi.URLParam(r, "userId")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
