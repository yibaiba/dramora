CREATE TABLE IF NOT EXISTS provider_audit_events (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    action text NOT NULL,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_email text,
    capability text NOT NULL,
    provider_type text NOT NULL,
    model text,
    success boolean NOT NULL,
    message text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS provider_audit_events_org_created_idx
    ON provider_audit_events (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS provider_audit_events_capability_idx
    ON provider_audit_events (organization_id, capability);
