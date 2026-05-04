-- Short Videos: Store generated short video records and their state
CREATE TABLE short_videos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  template_id UUID NOT NULL REFERENCES short_video_templates(id) ON DELETE RESTRICT,
  
  -- User inputs
  parameters JSONB NOT NULL, -- e.g., { "productName": "iPhone 15", "price": "5999", "description": "..." }
  
  -- Generation state
  status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, generating, completed, failed
  error_message TEXT,
  
  -- Generation results (populated after successful generation)
  result JSONB, -- e.g., { "heyGenVideoURL": "...", "thumbURL": "...", "duration": 30, "sizeBytes": 15728640 }
  
  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  -- Version for optimistic locking (optional, useful for concurrent updates)
  version INT NOT NULL DEFAULT 1
);

-- Indexes for common queries
CREATE INDEX idx_short_videos_organization_id ON short_videos(organization_id);
CREATE INDEX idx_short_videos_template_id ON short_videos(template_id);
CREATE INDEX idx_short_videos_status ON short_videos(status);
CREATE INDEX idx_short_videos_created_at ON short_videos(created_at DESC);
