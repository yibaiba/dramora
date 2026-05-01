CREATE TABLE IF NOT EXISTS llm_telemetry_aggregate (
    scope TEXT NOT NULL,
    key TEXT NOT NULL,
    counter BIGINT NOT NULL DEFAULT 0,
    error_counter BIGINT NOT NULL DEFAULT 0,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope, key)
);
