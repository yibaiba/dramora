-- Rollback: Remove batch-related fields from short_videos
ALTER TABLE short_videos DROP COLUMN IF EXISTS batch_id CASCADE;
ALTER TABLE short_videos DROP COLUMN IF EXISTS retry_count;
ALTER TABLE short_videos DROP COLUMN IF EXISTS created_by_batch;
