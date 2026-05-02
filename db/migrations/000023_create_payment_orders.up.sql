CREATE TABLE payment_orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    provider TEXT NOT NULL,           -- "stripe" / "alipay" / "wechat"
    provider_session_id TEXT,         -- Stripe session_id or similar
    amount INT8 NOT NULL,             -- 金额（分）
    currency TEXT NOT NULL,           -- 货币 ("USD" / "CNY")
    status TEXT NOT NULL,             -- "pending" / "success" / "failed" / "cancelled"
    error_reason TEXT,
    wallet_snapshot_id TEXT,          -- 关联的钱包快照（成功后）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    
    UNIQUE(provider, provider_session_id)
);

CREATE INDEX ON payment_orders(user_id, created_at DESC);
CREATE INDEX ON payment_orders(organization_id, created_at DESC);
CREATE INDEX ON payment_orders(status, created_at DESC);
