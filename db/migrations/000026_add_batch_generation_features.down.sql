-- Drop indexes
DROP INDEX IF EXISTS idx_generation_jobs_priority;
DROP INDEX IF EXISTS idx_generation_jobs_parent_job;

-- Drop queue_status table
DROP TABLE IF EXISTS queue_status;

-- Remove batch generation feature columns from generation_jobs table
ALTER TABLE generation_jobs
  DROP COLUMN IF EXISTS priority,
  DROP COLUMN IF EXISTS retry_count,
  DROP COLUMN IF EXISTS parent_job_id;
