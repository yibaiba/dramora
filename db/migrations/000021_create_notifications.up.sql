-- +migrate Up
CREATE TABLE notifications (
  id TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  recipient_user_id TEXT, -- NULL = broadcast
  kind TEXT NOT NULL CHECK (kind IN ('wallet_credit', 'wallet_debit', 'invitation_created', 'invitation_resent', 'provider_config_save')),
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  metadata JSONB DEFAULT '{}',
  read_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_org_recipient ON notifications(organization_id, recipient_user_id, created_at DESC);
CREATE INDEX idx_notifications_org_unread ON notifications(organization_id, read_at) WHERE read_at IS NULL;

-- +migrate Down
DROP TABLE notifications;
