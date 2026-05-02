-- billing_reports: 清算报表元数据表
CREATE TABLE billing_reports (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    period_start BIGINT NOT NULL,       -- Unix timestamp
    period_end BIGINT NOT NULL,
    total_debit_amount BIGINT NOT NULL DEFAULT 0,
    total_credit_amount BIGINT NOT NULL DEFAULT 0,
    total_refund_amount BIGINT NOT NULL DEFAULT 0,
    total_adjust_amount BIGINT NOT NULL DEFAULT 0,
    net_amount BIGINT NOT NULL DEFAULT 0,
    
    -- 待结算统计
    pending_billing_count INT NOT NULL DEFAULT 0,
    pending_billing_amount BIGINT NOT NULL DEFAULT 0,
    resolved_billing_count INT NOT NULL DEFAULT 0,
    failed_billing_count INT NOT NULL DEFAULT 0,
    
    -- 元数据
    status TEXT NOT NULL DEFAULT 'draft',
    generated_at BIGINT NOT NULL,
    generated_by TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_billing_reports_org_period ON billing_reports(organization_id, period_start, period_end);
CREATE INDEX idx_billing_reports_status ON billing_reports(status);

-- billing_report_breakdowns: 按操作类型的成本明细表
CREATE TABLE billing_report_breakdowns (
    id BIGSERIAL PRIMARY KEY,
    report_id TEXT NOT NULL REFERENCES billing_reports(id) ON DELETE CASCADE,
    operation_type TEXT NOT NULL,
    unit_cost BIGINT NOT NULL,        -- 该周期的单位成本
    usage_count BIGINT NOT NULL DEFAULT 0,
    total_debit_amount BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_billing_report_breakdowns_report ON billing_report_breakdowns(report_id);
