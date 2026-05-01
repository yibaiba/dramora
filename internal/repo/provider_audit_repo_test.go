package repo

import (
	"context"
	"testing"
	"time"
)

func TestMemoryProviderAuditFiltersByActorEmail(t *testing.T) {
	t.Parallel()
	r := NewMemoryProviderAuditRepository()
	ctx := context.Background()
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, email := range []string{"alice@example.com", "Alice@Example.com", "bob@example.com"} {
		if _, err := r.AppendProviderAuditEvent(ctx, AppendProviderAuditParams{
			EventID:        "evt-" + string(rune('0'+i)),
			OrganizationID: "org-1",
			Action:         "save",
			ActorEmail:     email,
			Capability:     "chat",
			ProviderType:   "openai",
			Success:        true,
			CreatedAt:      base.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	page, err := r.ListProviderAuditEvents(ctx, ProviderAuditFilter{
		OrganizationID: "org-1",
		ActorEmails:    []string{"alice@example.com"},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Events) != 2 {
		t.Fatalf("expected 2 events for alice (case-insensitive), got %d", len(page.Events))
	}

	page2, err := r.ListProviderAuditEvents(ctx, ProviderAuditFilter{
		OrganizationID: "org-1",
		ActorEmails:    []string{"bob@example.com"},
	})
	if err != nil {
		t.Fatalf("list bob: %v", err)
	}
	if len(page2.Events) != 1 {
		t.Fatalf("expected 1 event for bob, got %d", len(page2.Events))
	}
}
