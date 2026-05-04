ALTER TABLE short_videos ADD COLUMN heygen_video_id VARCHAR(255);
ALTER TABLE short_videos ADD COLUMN generation_status VARCHAR(50) DEFAULT 'pending';
CREATE INDEX idx_short_videos_generation_status ON short_videos(generation_status);
