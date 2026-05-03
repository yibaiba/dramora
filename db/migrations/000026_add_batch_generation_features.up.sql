-- Add batch generation feature columns to generation_jobs table
ALTER TABLE generation_jobs
  ADD COLUMN priority INT NOT NULL DEFAULT 50,
  ADD COLUMN retry_count INT NOT NULL DEFAULT 0,
  ADD COLUMN parent_job_id UUID NULL REFERENCES generation_jobs(id) ON DELETE SET NULL;

-- Create queue_status table for pause/resume state
CREATE TABLE queue_status (
  episode_id UUID PRIMARY KEY REFERENCES episodes(id) ON DELETE CASCADE,
  paused BOOLEAN NOT NULL DEFAULT FALSE,
  paused_at TIMESTAMP NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add index for efficient priority queries
CREATE INDEX idx_generation_jobs_priority ON generation_jobs (episode_id, priority DESC, status);

-- Add index for retry queries
CREATE INDEX idx_generation_jobs_parent_job ON generation_jobs (parent_job_id);
