CREATE TABLE IF NOT EXISTS short_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(13) NOT NULL UNIQUE,        -- GIFT-XXXXXXXX
    short_code VARCHAR(16) NOT NULL UNIQUE, -- base62(id)
    organization_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS short_codes_short_code_idx
    ON short_codes (short_code);

CREATE INDEX IF NOT EXISTS short_codes_code_idx
    ON short_codes (code);

CREATE INDEX IF NOT EXISTS short_codes_org_idx
    ON short_codes (organization_id);
