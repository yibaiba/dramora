package repo

// SQLite-compatible queries.
// Differences from PostgreSQL: ? placeholders, no ::uuid/::jsonb casts,
// COALESCE for nullable TEXT, strftime for timestamps, json() for JSONB inserts.

const sqliteListProjectsSQL = `
SELECT id, organization_id, name, description, status, created_at, updated_at
FROM projects
WHERE organization_id = ?
ORDER BY created_at DESC, id
`

const sqliteCreateProjectSQL = `
INSERT INTO projects (id, organization_id, name, description, status)
VALUES (?, ?, ?, ?, ?)
`

const sqliteGetProjectSQL = `
SELECT id, organization_id, name, description, status, created_at, updated_at
FROM projects
WHERE id = ? AND organization_id = ?
`

const sqliteLookupProjectByIDSQL = `
SELECT id, organization_id, name, description, status, created_at, updated_at
FROM projects
WHERE id = ?
`

const sqliteListEpisodesSQL = `
SELECT id, project_id, number, title, status, created_at, updated_at
FROM episodes
WHERE project_id = ?
ORDER BY number, created_at, id
`

const sqliteCreateEpisodeSQL = `
INSERT INTO episodes (id, project_id, number, title, status)
VALUES (?, ?, ?, ?, ?)
`

const sqliteGetEpisodeSQL = `
SELECT id, project_id, number, title, status, created_at, updated_at
FROM episodes
WHERE id = ?
`

const sqliteCreateWorkflowRunSQL = `
INSERT INTO workflow_runs (id, project_id, episode_id, status, input)
VALUES (?, ?, ?, ?, '{"kind":"story_analysis"}')
`

const sqliteGetWorkflowRunSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), status, created_at, updated_at
FROM workflow_runs
WHERE id = ?
`

const sqliteLoadWorkflowCheckpointSQL = `
SELECT output
FROM workflow_runs
WHERE id = ?
`

const sqliteSaveWorkflowCheckpointSQL = `
UPDATE workflow_runs
SET output = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
WHERE id = ?
`

const sqliteCompleteWorkflowRunSQL = `
UPDATE workflow_runs
SET status = ?, finished_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
WHERE id = ?
`

const sqliteCreateStorySourceSQL = `
INSERT INTO story_sources (id, project_id, episode_id, source_type, title, content_text, language)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

const sqliteListStorySourcesSQL = `
SELECT id, project_id, episode_id, source_type, title, content_text, language, created_at, updated_at
FROM story_sources
WHERE episode_id = ?
ORDER BY created_at DESC, id DESC
`

const sqliteLatestStorySourceSQL = `
SELECT id, project_id, episode_id, source_type, title, content_text, language, created_at, updated_at
FROM story_sources
WHERE episode_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1
`

const sqliteCreateGenerationJobSQL = `
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const sqliteCreateGenerationJobWithParamsSQL = `
INSERT INTO generation_jobs (
    id, project_id, episode_id, workflow_run_id, request_key, provider, model, task_type, status, prompt, params
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (request_key) DO UPDATE SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteCreateGenerationJobEventSQL = `
INSERT INTO generation_job_events (generation_job_id, status, message)
VALUES (?, ?, ?)
`

const sqliteListGenerationJobEventsSQL = `
SELECT id, generation_job_id, status, message, created_at
FROM generation_job_events
WHERE generation_job_id = ?
ORDER BY created_at, id
`

const sqliteGetGenerationJobSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), COALESCE(workflow_run_id, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id, ''),
       created_at, updated_at
FROM generation_jobs
WHERE id = ?
`

const sqliteListGenerationJobsSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), COALESCE(workflow_run_id, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id, ''),
       created_at, updated_at
FROM generation_jobs
ORDER BY created_at DESC, id
LIMIT 100
`

const sqliteListGenerationJobsByStatusSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), COALESCE(workflow_run_id, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id, ''),
       created_at, updated_at
FROM generation_jobs
WHERE status = ?
ORDER BY created_at, id
LIMIT ?
`

const sqliteAdvanceGenerationJobStatusSQL = `
UPDATE generation_jobs
SET status = ?,
    provider_task_id = COALESCE(NULLIF(?, ''), provider_task_id),
    result_asset_id = COALESCE(NULLIF(?, ''), result_asset_id),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?
  AND status = ?
`

const sqliteListApprovalGatesSQL = `
SELECT id, project_id, episode_id, COALESCE(workflow_run_id, ''),
       gate_type, subject_type, subject_id, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'), created_at, updated_at
FROM approval_gates
WHERE episode_id = ?
ORDER BY created_at, gate_type, id
`

const sqliteGetApprovalGateSQL = `
SELECT id, project_id, episode_id, COALESCE(workflow_run_id, ''),
       gate_type, subject_type, subject_id, status, reviewed_by, review_note,
       COALESCE(reviewed_at, '0001-01-01T00:00:00Z'), created_at, updated_at
FROM approval_gates
WHERE id = ?
`

const sqliteUpsertApprovalGateSQL = `
INSERT INTO approval_gates (
    id, project_id, episode_id, workflow_run_id, gate_type, subject_type, subject_id, status
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (episode_id, gate_type, subject_type, subject_id) DO UPDATE
SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteReviewApprovalGateSQL = `
UPDATE approval_gates
SET status = ?,
    reviewed_by = ?,
    review_note = ?,
    reviewed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?
`

const sqliteCreateStoryAnalysisSQL = `
INSERT INTO story_analyses (
    id, project_id, episode_id, story_source_id, workflow_run_id, generation_job_id, version,
    status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs
)
VALUES (
    ?, ?, ?, ?, ?, ?,
    COALESCE((SELECT MAX(version) + 1 FROM story_analyses WHERE episode_id = ?), 1),
    ?, ?, ?, ?, ?, ?, ?, ?
)
`

const sqliteListStoryAnalysesSQL = `
SELECT id, project_id, episode_id,
       COALESCE(story_source_id, ''),
       COALESCE(workflow_run_id, ''), COALESCE(generation_job_id, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs,
       created_at, updated_at
FROM story_analyses
WHERE episode_id = ?
ORDER BY version DESC, created_at DESC
`

const sqliteGetStoryAnalysisSQL = `
SELECT id, project_id, episode_id,
       COALESCE(story_source_id, ''),
       COALESCE(workflow_run_id, ''), COALESCE(generation_job_id, ''),
       version, status, summary, themes, character_seeds, scene_seeds, prop_seeds, outline, agent_outputs,
       created_at, updated_at
FROM story_analyses
WHERE id = ?
`

const sqliteUpsertCharacterSQL = `
INSERT INTO characters (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = excluded.story_analysis_id,
    name = excluded.name,
    description = excluded.description,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteUpsertSceneSQL = `
INSERT INTO scenes (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = excluded.story_analysis_id,
    name = excluded.name,
    description = excluded.description,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteUpsertPropSQL = `
INSERT INTO props (id, project_id, episode_id, story_analysis_id, code, name, description)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = excluded.story_analysis_id,
    name = excluded.name,
    description = excluded.description,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteListCharactersSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       code, name, description, COALESCE(character_bible, '{}'), created_at, updated_at
FROM characters
WHERE episode_id = ?
ORDER BY code
`

const sqliteGetCharacterSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       code, name, description, COALESCE(character_bible, '{}'), created_at, updated_at
FROM characters
WHERE id = ?
`

const sqliteSaveCharacterBibleSQL = `
UPDATE characters
SET character_bible = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?
RETURNING id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       code, name, description, COALESCE(character_bible, '{}'), created_at, updated_at
`

const sqliteListScenesSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       code, name, description, created_at, updated_at
FROM scenes
WHERE episode_id = ?
ORDER BY code
`

const sqliteListPropsSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       code, name, description, created_at, updated_at
FROM props
WHERE episode_id = ?
ORDER BY code
`

const sqliteCreateAssetSQL = `
INSERT INTO assets (id, project_id, episode_id, kind, purpose, uri, status)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE NOT EXISTS (
    SELECT 1 FROM assets WHERE episode_id = ?3 AND kind = ?4 AND purpose = ?5 AND uri = ?6
)
`

const sqliteGetExistingAssetSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), kind, purpose, uri, status, created_at, updated_at
FROM assets
WHERE episode_id = ? AND kind = ? AND purpose = ? AND uri = ?
LIMIT 1
`

const sqliteGetAssetSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), kind, purpose, uri, status, created_at, updated_at
FROM assets
WHERE id = ?
`

const sqliteListEpisodeAssetsSQL = `
SELECT id, project_id, COALESCE(episode_id, ''), kind, purpose, uri, status, created_at, updated_at
FROM assets
WHERE episode_id = ?
ORDER BY kind, purpose, created_at
`

const sqliteLockAssetSQL = `
UPDATE assets
SET status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?
`

const sqliteUpsertStoryboardShotSQL = `
INSERT INTO storyboard_shots (
    id, project_id, episode_id, story_analysis_id, scene_id, code, title, description, prompt, position, duration_ms
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (episode_id, code) DO UPDATE
SET story_analysis_id = excluded.story_analysis_id,
    scene_id = excluded.scene_id,
    title = excluded.title,
    description = excluded.description,
    prompt = excluded.prompt,
    position = excluded.position,
    duration_ms = excluded.duration_ms,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteListStoryboardShotsSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       COALESCE(scene_id, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE episode_id = ?
ORDER BY position, code
`

const sqliteGetStoryboardShotSQL = `
SELECT id, project_id, episode_id, COALESCE(story_analysis_id, ''),
       COALESCE(scene_id, ''), code, title, description, prompt, position, duration_ms, created_at, updated_at
FROM storyboard_shots
WHERE id = ?
`

const sqliteUpsertShotPromptPackSQL = `
INSERT INTO shot_prompt_packs (
    id, project_id, episode_id, shot_id, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (shot_id, preset) DO UPDATE
SET provider = excluded.provider,
    model = excluded.model,
    task_type = excluded.task_type,
    direct_prompt = excluded.direct_prompt,
    negative_prompt = excluded.negative_prompt,
    time_slices = excluded.time_slices,
    reference_bindings = excluded.reference_bindings,
    params = excluded.params,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteGetShotPromptPackSQL = `
SELECT id, project_id, episode_id, shot_id, provider, model, preset, task_type,
    direct_prompt, negative_prompt, time_slices, reference_bindings, params, created_at, updated_at
FROM shot_prompt_packs
WHERE shot_id = ?
ORDER BY updated_at DESC
LIMIT 1
`

const sqliteGetEpisodeTimelineSQL = `
SELECT id, episode_id, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = ?
`

const sqliteSaveEpisodeTimelineSQL = `
INSERT INTO timelines (id, episode_id, status, duration_ms)
VALUES (?, ?, ?, ?)
ON CONFLICT (episode_id) DO UPDATE
SET status = excluded.status,
    duration_ms = excluded.duration_ms,
    version = timelines.version + 1,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
`

const sqliteListTimelineTracksSQL = `
SELECT id, timeline_id, kind, name, position, created_at, updated_at
FROM timeline_tracks
WHERE timeline_id = ?
ORDER BY position, id
`

const sqliteListTimelineClipsSQL = `
SELECT id, timeline_id, track_id, COALESCE(asset_id, ''), kind,
       start_ms, duration_ms, trim_start_ms, created_at, updated_at
FROM timeline_clips
WHERE timeline_id = ?
ORDER BY start_ms, id
`

const sqliteDeleteTimelineTracksSQL = `
DELETE FROM timeline_tracks
WHERE timeline_id = ?
`

const sqliteCreateTimelineTrackSQL = `
INSERT INTO timeline_tracks (id, timeline_id, kind, name, position)
VALUES (?, ?, ?, ?, ?)
`

const sqliteCreateTimelineClipSQL = `
INSERT INTO timeline_clips (id, timeline_id, track_id, asset_id, kind, start_ms, duration_ms, trim_start_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

const sqliteCreateExportSQL = `
INSERT INTO exports (id, timeline_id, status, format)
VALUES (?, ?, ?, ?)
`

const sqliteGetExportSQL = `
SELECT id, timeline_id, status, format, created_at, updated_at
FROM exports
WHERE id = ?
`

const sqliteListExportsByStatusSQL = `
SELECT id, timeline_id, status, format, created_at, updated_at
FROM exports
WHERE status = ?
ORDER BY created_at, id
LIMIT ?
`

const sqliteAdvanceExportStatusSQL = `
UPDATE exports
SET status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?
  AND status = ?
`

const sqliteGetStorySourceByIDSQL = `
SELECT id, project_id, episode_id, source_type, title, content_text, language, created_at, updated_at
FROM story_sources
WHERE id = ?
`

const sqliteGetGenerationJobByRequestKeySQL = `
SELECT id, project_id, COALESCE(episode_id, ''), COALESCE(workflow_run_id, ''),
       provider, model, task_type, status, prompt, params, COALESCE(provider_task_id, ''),
       COALESCE(result_asset_id, ''),
       created_at, updated_at
FROM generation_jobs
WHERE request_key = ?
`

const sqliteGetTimelineByIDSQL = `
SELECT id, episode_id, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE id = ?
`

const sqliteGetTimelineByEpisodeForUpdateSQL = `
SELECT id, episode_id, status, version, duration_ms, created_at, updated_at
FROM timelines
WHERE episode_id = ?
`
