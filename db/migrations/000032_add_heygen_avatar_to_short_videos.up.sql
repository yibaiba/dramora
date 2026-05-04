-- Add heygen_avatar_id column to short_videos table
ALTER TABLE short_videos
ADD COLUMN heygen_avatar_id VARCHAR(50) NOT NULL DEFAULT 'avatar_001';

-- Add index on heygen_avatar_id for future queries
CREATE INDEX idx_short_videos_heygen_avatar_id ON short_videos(heygen_avatar_id);
