CREATE TABLE IF NOT EXISTS worker_metric_state (
    metric_kind TEXT PRIMARY KEY,
    counter BIGINT NOT NULL DEFAULT 0,
    last_reason TEXT,
    last_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
