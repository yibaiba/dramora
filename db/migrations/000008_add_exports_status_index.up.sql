CREATE INDEX idx_exports_status_created_at
    ON exports (status, created_at);
