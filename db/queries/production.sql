-- name: GetWorkflowRun :one
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), status, created_at, updated_at
FROM workflow_runs
WHERE id = $1::uuid;

-- name: ListGenerationJobs :many
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at
FROM generation_jobs
ORDER BY created_at DESC, id
LIMIT 100;

-- name: GetGenerationJob :one
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at
FROM generation_jobs
WHERE id = $1::uuid;

-- name: CreateGenerationJob :one
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10)
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at;

-- name: CreateGenerationJobWithParams :one
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt, params
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11::jsonb)
ON CONFLICT (request_key) DO UPDATE
SET updated_at = now()
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at;

-- name: ListGenerationJobsByStatus :many
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at
FROM generation_jobs
WHERE status = $1
ORDER BY created_at, id
LIMIT $2;

-- name: AdvanceGenerationJobStatus :one
UPDATE generation_jobs
SET status = $3,
    provider_task_id = COALESCE(NULLIF($4, ''), provider_task_id),
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), COALESCE(workflow_run_id::text, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       created_at, updated_at;

-- name: ListApprovalGates :many
SELECT id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
FROM approval_gates
WHERE episode_id = $1::uuid
ORDER BY created_at, gate_type, id;

-- name: GetApprovalGate :one
SELECT id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
FROM approval_gates
WHERE id = $1::uuid;

-- name: UpsertApprovalGate :one
INSERT INTO approval_gates (
    id, project_id, episode_id, workflow_run_id, gate_type, subject_type, subject_id, status
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::uuid, $8)
ON CONFLICT (episode_id, gate_type, subject_type, subject_id) DO UPDATE
SET updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at;

-- name: ReviewApprovalGate :one
UPDATE approval_gates
SET status = $2,
    reviewed_by = $3,
    review_note = $4,
    reviewed_at = now(),
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, project_id::text, episode_id::text, COALESCE(workflow_run_id::text, ''),
       gate_type, subject_type, subject_id::text, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at;

-- name: CreateStoryAnalysis :one
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
    created_at, updated_at;

-- name: ListStoryAnalyses :many
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds,
       created_at, updated_at
FROM story_analyses
WHERE episode_id = $1::uuid
ORDER BY version DESC, created_at DESC;

-- name: GetStoryAnalysis :one
SELECT id::text, project_id::text, episode_id::text,
       COALESCE(workflow_run_id::text, ''), COALESCE(generation_job_id::text, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds,
       created_at, updated_at
FROM story_analyses
WHERE id = $1::uuid;

-- name: GetEpisodeTimeline :one
SELECT id::text, episode_id::text, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = $1::uuid;

-- name: SaveEpisodeTimeline :one
INSERT INTO timelines (id, episode_id, status, duration_ms)
VALUES ($1::uuid, $2::uuid, $3, $4)
ON CONFLICT (episode_id) DO UPDATE
SET status = EXCLUDED.status,
    duration_ms = EXCLUDED.duration_ms,
    version = timelines.version + 1,
    updated_at = now()
RETURNING id::text, episode_id::text, status, version, duration_ms, created_at, updated_at;

-- name: UpsertCharacter :one
INSERT INTO characters (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at;

-- name: UpsertScene :one
INSERT INTO scenes (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at;

-- name: UpsertProp :one
INSERT INTO props (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = EXCLUDED.story_analysis_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at;

-- name: ListCharacters :many
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM characters
WHERE episode_id = $1::uuid
ORDER BY code;

-- name: ListScenes :many
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM scenes
WHERE episode_id = $1::uuid
ORDER BY code;

-- name: ListProps :many
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       code, name, description, created_at, updated_at
FROM props
WHERE episode_id = $1::uuid
ORDER BY code;

-- name: CreateAsset :one
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
LIMIT 1;

-- name: ListEpisodeAssets :many
SELECT id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at
FROM assets
WHERE episode_id = $1::uuid
ORDER BY kind, purpose, created_at;

-- name: LockAsset :one
UPDATE assets
SET status = $2,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, project_id::text, COALESCE(episode_id::text, ''), kind, purpose, uri, status, created_at, updated_at;

-- name: ListStoryboardShots :many
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE episode_id = $1::uuid
ORDER BY position, code;

-- name: GetStoryboardShot :one
SELECT id::text, project_id::text, episode_id::text, COALESCE(story_analysis_id::text, ''),
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE id = $1::uuid;

-- name: UpsertStoryboardShot :one
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
       COALESCE(scene_id::text, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at;

-- name: UpsertShotPromptPack :one
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
    direct_prompt, negative_prompt, time_slices, reference_bindings, params, created_at, updated_at;

-- name: GetShotPromptPack :one
SELECT id::text, project_id::text, episode_id::text, shot_id::text, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params, created_at, updated_at
FROM shot_prompt_packs
WHERE shot_id = $1::uuid
ORDER BY updated_at DESC
LIMIT 1;

-- name: ListTimelineTracks :many
SELECT id::text, timeline_id::text, kind, name, position, created_at, updated_at
FROM timeline_tracks
WHERE timeline_id = $1::uuid
ORDER BY position, id;

-- name: ListTimelineClips :many
SELECT id::text, timeline_id::text, track_id::text, COALESCE(asset_id::text, ''), kind,
       start_ms, duration_ms, trim_start_ms, created_at, updated_at
FROM timeline_clips
WHERE timeline_id = $1::uuid
ORDER BY start_ms, id;

-- name: CreateTimelineTrack :one
INSERT INTO timeline_tracks (id, timeline_id, kind, name, position)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, timeline_id::text, kind, name, position, created_at, updated_at;

-- name: CreateTimelineClip :one
INSERT INTO timeline_clips (id, timeline_id, track_id, asset_id, kind, start_ms, duration_ms, trim_start_ms)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
RETURNING id::text, timeline_id::text, track_id::text, COALESCE(asset_id::text, ''), kind,
       start_ms, duration_ms, trim_start_ms, created_at, updated_at;

-- name: CreateExport :one
INSERT INTO exports (id, timeline_id, status, format)
VALUES ($1::uuid, $2::uuid, $3, $4)
RETURNING id::text, timeline_id::text, status, format, created_at, updated_at;

-- name: GetExport :one
SELECT id::text, timeline_id::text, status, format, created_at, updated_at
FROM exports
WHERE id = $1::uuid;

-- name: ListExportsByStatus :many
SELECT id::text, timeline_id::text, status, format, created_at, updated_at
FROM exports
WHERE status = $1
ORDER BY created_at, id
LIMIT $2;

-- name: AdvanceExportStatus :one
UPDATE exports
SET status = $3,
    updated_at = now()
WHERE id = $1::uuid
  AND status = $2
RETURNING id::text, timeline_id::text, status, format, created_at, updated_at;
