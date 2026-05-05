-- Create batch_submissions table for managing batch video generation operations
CREATE TABLE batch_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  total_count INT NOT NULL DEFAULT 0,
  completed_count INT NOT NULL DEFAULT 0,
  failed_count INT NOT NULL DEFAULT 0,
  cancelled_count INT NOT NULL DEFAULT 0,
  status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, queued, processing, completed, cancelled, failed
  concurrency_limit INT NOT NULL DEFAULT 4,
  retry_limit INT NOT NULL DEFAULT 3,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX idx_batch_submissions_org_id ON batch_submissions(organization_id);
CREATE INDEX idx_batch_submissions_created_by ON batch_submissions(created_by_user_id);
CREATE INDEX idx_batch_submissions_status ON batch_submissions(status);
CREATE INDEX idx_batch_submissions_created_at ON batch_submissions(created_at DESC);
