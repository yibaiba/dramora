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

const createStorySourceSQL = `
INSERT INTO story_sources (id, project_id, episode_id, source_type, title, content_text, language)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
RETURNING id::text, project_id::text, episode_id::text, source_type, title, content_text, language, created_at, updated_at
`

const listStorySourcesSQL = `
SELECT id::text, project_id::text, episode_id::text, source_type, title, content_text, language, created_at, updated_at
FROM story_sources
WHERE episode_id = $1::uuid
ORDER BY created_at DESC, id DESC
`

const latestStorySourceSQL = `
SELECT id::text, project_id::text, episode_id::text, source_type, title, content_text, language, created_at, updated_at
FROM story_sources
WHERE episode_id = $1::uuid
ORDER BY created_at DESC, id DESC
LIMIT 1
`

const createGenerationJobSQL = `
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10)
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
`

const createGenerationJobEventSQL = `
INSERT INTO generation_job_events (generation_job_id, status, message)
VALUES ($1::uuid, $2, $3)
`

const listGenerationJobsSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
FROM generation_jobs
ORDER BY created_at DESC, id
LIMIT 100
`

const getGenerationJobSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
FROM generation_jobs
WHERE id = $1::uuid
`

const createGenerationJobWithParamsSQL = `
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt, params
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11::jsonb)
ON CONFLICT (request_key) DO UPDATE
SET updated_at = now()
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
`

const listGenerationJobsByStatusSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
FROM generation_jobs
WHERE status = $1
ORDER BY created_at, id
LIMIT $2
`

const advanceGenerationJobStatusSQL = `
UPDATE generation_jobs
SET status = $3,
    provider_task_id = COALESCE(NULLIF($4, ''), provider_task_id),
    result_asset_id = COALESCE(NULLIF($5, '')::uuid, result_asset_id),
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id::text, ''),
       created_at, updated_at
`

const listApprovalGatesSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
FROM approval_gates
WHERE episode_id = $1::uuid
ORDER BY created_at, gate_type, id
`

const getApprovalGateSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
FROM approval_gates
WHERE id = $1::uuid
`

const upsertApprovalGateSQL = `
INSERT INTO approval_gates (
    id, project_id, episode_id, workflow_run_id, gate_type, subject_type, subject_id, status
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::uuid, $8)
ON CONFLICT (episode_id, gate_type, subject_type, subject_id) DO UPDATE
SET updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
`

const reviewApprovalGateSQL = `
UPDATE approval_gates
SET status = $2,
    reviewed_by = $3,
    review_note = $4,
    reviewed_at = now(),
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
`

const createStoryAnalysisSQL = `
INSERT INTO story_analyses (
    id, project_id, episode_id, story_source_id, workflow_run_id, generation_job_id, version,
    status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs
)
VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6::uuid,
    COALESCE((SELECT MAX(version) + 1 FROM story_analyses WHERE episode_id = $3::uuid), 1),
    $7, $8, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb
)
RETURNING id::text, project_id::text, episode_id::text,
    COALESCE(story_source_id::text, ''),
    COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
    version, status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs,
    created_at, updated_at
`

const listStoryAnalysesSQL = `
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(story_source_id::text, ''),
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs,
       created_at, updated_at
FROM story_analyses
WHERE episode_id = $1::uuid
ORDER BY version DESC, created_at DESC
`

const getStoryAnalysisSQL = `
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(story_source_id::text, ''),
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs,
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

const createAssetSQL = `
WITH existing AS (
    SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at
    FROM assets
    WHERE episode_id = $3::uuid
      AND kind = $4
      AND purpose = $5
      AND uri = $6
),
inserted AS (
    INSERT INTO assets (id, project_id, episode_id, kind, purpose, uri, status)
    SELECT $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7
    WHERE NOT EXISTS (SELECT 1 FROM existing)
    RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at
)
SELECT * FROM inserted
UNION ALL
SELECT * FROM existing
LIMIT 1
`

const listEpisodeAssetsSQL = `
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at
FROM assets
WHERE episode_id = $1::uuid
ORDER BY kind, purpose, created_at
`

const lockAssetSQL = `
UPDATE assets
SET status = $2,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at
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

const getStoryboardShotSQL = `
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE id = $1::uuid
`

const upsertShotPromptPackSQL = `
INSERT INTO shot_prompt_packs (
    id, project_id, episode_id, shot_id, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb)
ON CONFLICT (shot_id, preset) DO UPDATE
SET provider = EXCLUDED.provider,
    model = EXCLUDED.model,
    task_type = EXCLUDED.task_type,
    direct_prompt = EXCLUDED.direct_prompt,
    negative_prompt = EXCLUDED.negative_prompt,
    time_slices = EXCLUDED.time_slices,
    reference_bindings = EXCLUDED.reference_bindings,
    params = EXCLUDED.params,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, shot_id::text, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params, created_at, updated_at
`

const getShotPromptPackSQL = `
SELECT id::text, project_id::text, episode_id::text, shot_id::text, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params, created_at, updated_at
FROM shot_prompt_packs
WHERE shot_id = $1::uuid
ORDER BY updated_at DESC
LIMIT 1
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

const listExportsByStatusSQL = `
SELECT id::text, timeline_id::text, status, format, created_at, updated_at
FROM exports
WHERE status = $1
ORDER BY created_at, id
LIMIT $2
`

const advanceExportStatusSQL = `
UPDATE exports
SET status = $3,
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, timeline_id::text, status, format, created_at, updated_at
`
