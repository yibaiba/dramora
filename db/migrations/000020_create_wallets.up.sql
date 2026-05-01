CREATE TABLE IF NOT EXISTS wallets (
    organization_id TEXT PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    direction INTEGER NOT NULL DEFAULT 1,
    amount BIGINT NOT NULL,
    reason TEXT,
    ref_type TEXT,
    ref_id TEXT,
    balance_after BIGINT NOT NULL,
    actor_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS wallet_transactions_org_created_idx
    ON wallet_transactions (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS wallet_transactions_ref_idx
    ON wallet_transactions (organization_id, ref_type, ref_id);
