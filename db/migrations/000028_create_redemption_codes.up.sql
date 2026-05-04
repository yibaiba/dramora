CREATE TABLE IF NOT EXISTS redemption_campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',  -- active / inactive / expired
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    organization_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS redemption_campaigns_org_status_idx
    ON redemption_campaigns (organization_id, status);

CREATE TABLE IF NOT EXISTS redemption_codes (
    id TEXT PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,  -- e.g., GIFT-XXXXX, case-insensitive
    organization_id TEXT NOT NULL,
    campaign_id TEXT,  -- nullable for backward compatibility
    amount BIGINT NOT NULL,  -- amount in credits/積分
    status TEXT NOT NULL DEFAULT 'unused',  -- unused / used
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_by TEXT,  -- user_id of who redeemed it
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,  -- null = never expires
    reason TEXT,  -- e.g., "monthly reward", "partner gift"
    version INT NOT NULL DEFAULT 0  -- optimistic lock version
);

CREATE INDEX IF NOT EXISTS redemption_codes_code_idx
    ON redemption_codes (code);

CREATE INDEX IF NOT EXISTS redemption_codes_org_status_idx
    ON redemption_codes (organization_id, status);

CREATE INDEX IF NOT EXISTS redemption_codes_campaign_idx
    ON redemption_codes (campaign_id, status);

CREATE INDEX IF NOT EXISTS redemption_codes_org_created_idx
    ON redemption_codes (organization_id, created_at DESC);
