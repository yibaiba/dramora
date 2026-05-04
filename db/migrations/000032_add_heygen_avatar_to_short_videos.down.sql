-- Remove heygen_avatar_id column from short_videos table
ALTER TABLE short_videos
DROP INDEX IF EXISTS idx_short_videos_heygen_avatar_id,
DROP COLUMN heygen_avatar_id;
