package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type providerAuditEventDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Action         string `json:"action"`
	ActorUserID    string `json:"actor_user_id,omitempty"`
	ActorEmail     string `json:"actor_email,omitempty"`
	Capability     string `json:"capability"`
	ProviderType   string `json:"provider_type"`
	Model          string `json:"model,omitempty"`
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func providerAuditEventToDTO(ev domain.ProviderAuditEvent) providerAuditEventDTO {
	return providerAuditEventDTO{
		ID:             ev.ID,
		OrganizationID: ev.OrganizationID,
		Action:         ev.Action,
		ActorUserID:    ev.ActorUserID,
		ActorEmail:     ev.ActorEmail,
		Capability:     ev.Capability,
		ProviderType:   ev.ProviderType,
		Model:          ev.Model,
		Success:        ev.Success,
		Message:        ev.Message,
		CreatedAt:      ev.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (a *api) listProviderAuditEvents(w http.ResponseWriter, r *http.Request) {
	if a.providerService == nil {
		writeJSON(w, http.StatusOK, Envelope{"events": []providerAuditEventDTO{}, "has_more": false})
		return
	}
	q := r.URL.Query()
	filter := repo.ProviderAuditFilter{}
	if actions := q.Get("action"); actions != "" {
		filter.Actions = splitCSV(actions)
	}
	if caps := q.Get("capability"); caps != "" {
		filter.Capabilities = splitCSV(caps)
	}
	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = &t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = &t
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	page, err := a.providerService.ListProviderAuditEvents(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]providerAuditEventDTO, len(page.Events))
	for i, ev := range page.Events {
		out[i] = providerAuditEventToDTO(ev)
	}
	writeJSON(w, http.StatusOK, Envelope{"events": out, "has_more": page.HasMore})
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
