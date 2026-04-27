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

const createStoryAnalysisSQL = `
INSERT INTO story_analyses (
    id, project_id, episode_id, workflow_run_id, generation_job_id, version,
    status, summary, themes, character_seeds, scene_seeds, prop_seeds
)
VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid,
    COALESCE((SELECT MAX(version) + 1 FROM story_analyses WHERE episode_id = $3::uuid), 1),
    $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb
)
RETURNING id::text, project_id::text, episode_id::text,
    COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
    version, status, summary, themes, character_seeds, scene_seeds, prop_seeds,
    created_at, updated_at
`

const listStoryAnalysesSQL = `
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds,
       created_at, updated_at
FROM story_analyses
WHERE episode_id = $1::uuid
ORDER BY version DESC, created_at DESC
`

const getStoryAnalysisSQL = `
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds,
       created_at, updated_at
FROM story_analyses
WHERE id = $1::uuid
`

const getEpisodeTimelineSQL = `
SELECT id::text, episode_id::text, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = $1::uuid
`

const listTimelineTracksSQL = `
SELECT id::text, timeline_id::text, kind, name, position, created_at, updated_at
FROM timeline_tracks
WHERE timeline_id = $1::uuid
ORDER BY position, id
`

const listTimelineClipsSQL = `
SELECT id::text, timeline_id::text, track_id::text, COALESCE(asset_id::text, ''), kind,
       start_ms, duration_ms, trim_start_ms, created_at, updated_at
FROM timeline_clips
WHERE timeline_id = $1::uuid
ORDER BY start_ms, id
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

const deleteTimelineTracksSQL = `
DELETE FROM timeline_tracks
WHERE timeline_id = $1::uuid
`

const createTimelineTrackSQL = `
INSERT INTO timeline_tracks (id, timeline_id, kind, name, position)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, timeline_id::text, kind, name, position, created_at, updated_at
`

const createTimelineClipSQL = `
INSERT INTO timeline_clips (id, timeline_id, track_id, asset_id, kind, start_ms, duration_ms, trim_start_ms)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
RETURNING id::text, timeline_id::text, track_id::text, COALESCE(asset_id::text, ''), kind,
       start_ms, duration_ms, trim_start_ms, created_at, updated_at
`

const upsertCharacterSQL = `
INSERT INTO characters (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
`

const upsertSceneSQL = `
INSERT INTO scenes (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
`

const upsertPropSQL = `
INSERT INTO props (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
`

const listCharactersSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM characters
WHERE episode_id = $1::uuid
ORDER BY code
`

const listScenesSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM scenes
WHERE episode_id = $1::uuid
ORDER BY code
`

const listPropsSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM props
WHERE episode_id = $1::uuid
ORDER BY code
`

const upsertStoryboardShotSQL = `
INSERT INTO storyboard_shots (
    id, project_id, episode_id, story_analysis_id, scene_id, code, title, description, prompt, position, duration_ms
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7, $8, $9, $10, $11)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    scene_id = EXCLUDED.scene_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    prompt = EXCLUDED.prompt,
    position = EXCLUDED.position,
    duration_ms = EXCLUDED.duration_ms,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
`

const listStoryboardShotsSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE episode_id = $1::uuid
ORDER BY position, code
`

const createExportSQL = `
INSERT INTO exports (id, timeline_id, status, format)
VALUES ($1::uuid, $2::uuid, $3, $4)
RETURNING id::text, timeline_id::text, status, format, created_at, updated_at
`

const getExportSQL = `
SELECT id::text, timeline_id::text, status, format, created_at, updated_at
FROM exports
WHERE id = $1::uuid
`
