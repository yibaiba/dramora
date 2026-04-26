package repo

const getWorkflowRunSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), status, created_at, updated_at
FROM workflow_runs
WHERE id = $1::uuid
`

const createWorkflowRunSQL = `
INSERT INTO workflow_runs (id, project_id, episode_id, status, input)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, '{"kind":"story_analysis"}'::jsonb)
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), status, created_at, updated_at
`

const createGenerationJobSQL = `
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10)
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
`

const createGenerationJobEventSQL = `
INSERT INTO generation_job_events (generation_job_id, status, message)
VALUES ($1::uuid, $2, $3)
`

const listGenerationJobsSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
ORDER BY created_at DESC, id
LIMIT 100
`

const getGenerationJobSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
WHERE id = $1::uuid
`

const listGenerationJobsByStatusSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
FROM generation_jobs
WHERE status = $1
ORDER BY created_at, id
LIMIT $2
`

const advanceGenerationJobStatusSQL = `
UPDATE generation_jobs
SET status = $3,
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, created_at, updated_at
`

const getEpisodeTimelineSQL = `
SELECT id::text, episode_id::text, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = $1::uuid
`

const saveEpisodeTimelineSQL = `
INSERT INTO timelines (id, episode_id, status, duration_ms)
VALUES ($1::uuid, $2::uuid, $3, $4)
ON CONFLICT (episode_id) DO UPDATE
SET status = EXCLUDED.status,
    duration_ms = EXCLUDED.duration_ms,
    version = timelines.version + 1,
    updated_at = now()
RETURNING id::text, episode_id::text, status, version, duration_ms, created_at, updated_at
`
