CREATE TABLE organization_invitation_events (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invitation_id uuid NOT NULL,
    action text NOT NULL,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_email text,
    email text NOT NULL,
    role text NOT NULL,
    note text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX organization_invitation_events_org_created_idx
    ON organization_invitation_events (organization_id, created_at DESC);

CREATE INDEX organization_invitation_events_invitation_idx
    ON organization_invitation_events (invitation_id);
