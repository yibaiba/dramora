DROP INDEX IF EXISTS idx_short_videos_generation_status;
ALTER TABLE short_videos DROP COLUMN IF EXISTS generation_status;
ALTER TABLE short_videos DROP COLUMN IF EXISTS heygen_video_id;
