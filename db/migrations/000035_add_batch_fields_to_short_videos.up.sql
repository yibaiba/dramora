-- Add batch-related fields to short_videos table
ALTER TABLE short_videos ADD COLUMN batch_id UUID REFERENCES batch_submissions(id) ON DELETE SET NULL;
ALTER TABLE short_videos ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE short_videos ADD COLUMN created_by_batch BOOLEAN NOT NULL DEFAULT FALSE;

-- Create indexes for batch operations
CREATE INDEX idx_short_videos_batch_id ON short_videos(batch_id);
CREATE INDEX idx_short_videos_retry_count ON short_videos(retry_count);
