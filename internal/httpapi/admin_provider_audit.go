package httpapi

import (
	"encoding/csv"
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
	if actor := q.Get("actor"); actor != "" {
		filter.ActorEmails = splitCSV(actor)
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
	if strings.EqualFold(q.Get("format"), "csv") {
		writeProviderAuditCSV(w, page.Events)
		return
	}
	out := make([]providerAuditEventDTO, len(page.Events))
	for i, ev := range page.Events {
		out[i] = providerAuditEventToDTO(ev)
	}
	writeJSON(w, http.StatusOK, Envelope{"events": out, "has_more": page.HasMore})
}

func writeProviderAuditCSV(w http.ResponseWriter, events []domain.ProviderAuditEvent) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="provider-audit.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"id", "organization_id", "action", "actor_user_id", "actor_email",
		"capability", "provider_type", "model", "success", "message", "created_at",
	})
	for _, ev := range events {
		_ = cw.Write([]string{
			ev.ID, ev.OrganizationID, ev.Action, ev.ActorUserID, ev.ActorEmail,
			ev.Capability, ev.ProviderType, ev.Model,
			strconv.FormatBool(ev.Success), ev.Message,
			ev.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
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
