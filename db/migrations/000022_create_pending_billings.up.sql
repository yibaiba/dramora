CREATE TABLE IF NOT EXISTS pending_billings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    operation_type VARCHAR(50) NOT NULL,
    ref_type VARCHAR(100) NOT NULL,
    ref_id VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    last_error_msg TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_pending_billings_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    
    CONSTRAINT unique_pending_billing_ref 
        UNIQUE (organization_id, operation_type, ref_type, ref_id)
);

-- Index for worker to fetch pending retries
CREATE INDEX idx_pending_billings_status_updated 
    ON pending_billings(status, updated_at)
    WHERE status IN ('pending', 'retrying');

-- Index for checking duplicate operations (idempotency)
CREATE INDEX idx_pending_billings_ref 
    ON pending_billings(organization_id, ref_type, ref_id);
