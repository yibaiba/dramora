-- name: GetWorkflowRun :one
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), status, created_at, updated_at
FROM workflow_runs
WHERE id = $1::uuid;

-- name: ListGenerationJobs :many
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
ORDER BY created_at DESC, id
LIMIT 100;

-- name: GetGenerationJob :one
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
WHERE id = $1::uuid;

-- name: ListGenerationJobsByStatus :many
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
WHERE status = $1
ORDER BY created_at, id
LIMIT $2;

-- name: AdvanceGenerationJobStatus :one
UPDATE generation_jobs
SET status = $3,
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at;

-- name: GetEpisodeTimeline :one
SELECT id::text, episode_id::text, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = $1::uuid;
