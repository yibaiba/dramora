CREATE TABLE IF NOT EXISTS llm_telemetry_daily (
    scope TEXT NOT NULL,
    key TEXT NOT NULL,
    day_utc DATE NOT NULL,
    counter BIGINT NOT NULL DEFAULT 0,
    error_counter BIGINT NOT NULL DEFAULT 0,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope, key, day_utc)
);
CREATE INDEX IF NOT EXISTS llm_telemetry_daily_day_idx ON llm_telemetry_daily (day_utc);
