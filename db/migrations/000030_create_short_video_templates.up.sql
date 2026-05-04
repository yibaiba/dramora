-- Short Video Templates: Store template configurations for AI-generated e-commerce videos
CREATE TABLE short_video_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  
  -- Basic metadata
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category VARCHAR(100) NOT NULL, -- e.g., 'product-focus', 'promotion', 'usage-scenario'
  
  -- Template configuration (JSON schema for template parameters)
  config JSONB NOT NULL,
  
  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  -- Constraints
  UNIQUE(organization_id, name)
);

-- Indexes for common queries
CREATE INDEX idx_short_video_templates_organization_id ON short_video_templates(organization_id);
CREATE INDEX idx_short_video_templates_category ON short_video_templates(category);
