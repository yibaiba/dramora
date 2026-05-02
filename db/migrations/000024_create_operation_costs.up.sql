CREATE TABLE IF NOT EXISTS operation_costs (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(50) NOT NULL,
    organization_id VARCHAR(255) DEFAULT '00000000-0000-0000-0000-000000000001',
    credits_cost INT NOT NULL,
    effective_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);

CREATE UNIQUE INDEX idx_operation_costs_type_org_effective ON operation_costs(
    operation_type,
    organization_id,
    effective_at
) WHERE expires_at IS NULL;

CREATE INDEX idx_operation_costs_lookup ON operation_costs(
    operation_type,
    organization_id,
    effective_at
);

CREATE TABLE IF NOT EXISTS operation_cost_history (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(50) NOT NULL,
    organization_id VARCHAR(255) DEFAULT '00000000-0000-0000-0000-000000000001',
    old_cost INT,
    new_cost INT,
    effective_at TIMESTAMP NOT NULL,
    reason TEXT,
    changed_by VARCHAR(255) NOT NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_cost_history_lookup ON operation_cost_history(
    operation_type,
    organization_id,
    changed_at
);
