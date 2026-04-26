# Go Backend Domain Model and PostgreSQL Schema Draft

## Purpose

Define Manmu's first backend domain model for a Go + PostgreSQL modular monolith.

This is a planning schema, not a final migration. It is designed around:

- AI manju production,
- story/character/scene/prop/shot assets,
- multi-agent workflow,
- provider-agnostic model calls,
- online timeline editing and export,
- traceability, approval, and cost control.

## Backend shape

Recommended first architecture:

```text
apps/api           Go HTTP API
apps/worker        Go async workers
internal/domain    Pure domain structs and enums
internal/service   Use cases / application services
internal/repo      PostgreSQL repositories
internal/provider  LLM/image/video/audio provider adapters
internal/workflow  Workflow graph and agent orchestration
internal/media     Object storage, thumbnails, FFmpeg/export helpers
internal/auth      users, orgs, memberships, permissions
```

Use PostgreSQL as source of truth, River/PostgreSQL transactional jobs as the MVP queue default, Redis as optional cache/realtime/rate-limit infrastructure, and S3-compatible object storage for media.

## Naming conventions

- Tables: plural snake_case.
- Primary keys: `id uuid primary key`.
- Foreign keys: `<entity>_id uuid not null references <table>(id)`.
- Timestamps: `created_at`, `updated_at`; add `deleted_at` only when soft delete is required.
- JSON columns: `jsonb`, but only for flexible provider metadata, prompt params, and generated structured artifacts. Core query fields stay typed columns.
- All long-running states use explicit status enums or constrained text.

## Domain modules

```text
Identity / Workspace
Project / Story
Character / Scene / Prop
Shot / Storyboard
Asset / Artifact Lineage
Workflow / Agent
Model Provider / Generation Job
Timeline / Export
Review / Safety / Continuity
Cost / Audit
```

## 1. Identity and workspace

### users

Creator/admin account.

Important columns:

- `id`
- `email`
- `display_name`
- `avatar_asset_id`
- `role`
- `created_at`
- `updated_at`

### organizations

Workspace or studio.

Important columns:

- `id`
- `name`
- `slug`
- `plan`
- `created_at`
- `updated_at`

### organization_members

Membership and permissions.

Important columns:

- `organization_id`
- `user_id`
- `role`
- `created_at`

Unique:

- `(organization_id, user_id)`

## 2. Project and story

### projects

Top-level creative workspace.

Important columns:

- `id`
- `organization_id`
- `owner_user_id`
- `title`
- `slug`
- `description`
- `status`: `draft | active | archived`
- `visual_style`
- `target_aspect_ratio`: `16:9 | 9:16 | 1:1`
- `target_duration_sec`
- `model_budget_cents`
- `created_at`
- `updated_at`

### seasons

Optional grouping for longer IPs.

Important columns:

- `id`
- `project_id`
- `title`
- `sequence_no`
- `created_at`
- `updated_at`

### episodes

Production unit.

Important columns:

- `id`
- `project_id`
- `season_id`
- `title`
- `sequence_no`
- `status`
- `target_duration_sec`
- `summary`
- `created_at`
- `updated_at`

### story_sources

Original input: idea, outline, novel chapter, uploaded file, pasted script.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `source_type`: `idea | outline | novel | script | file | url`
- `title`
- `content_text`
- `source_asset_id`
- `language`
- `created_by`
- `created_at`

### story_analyses

Structured story analysis created by Story Analyst Agent.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `story_source_id`
- `version_no`
- `status`: `draft | approved | superseded`
- `summary`
- `theme`
- `conflict`
- `emotional_curve jsonb`
- `timeline jsonb`
- `raw_output jsonb`
- `created_by_agent_run_id`
- `approved_by`
- `approved_at`
- `created_at`

### world_bibles

Project-level worldbuilding.

Important columns:

- `id`
- `project_id`
- `version_no`
- `status`
- `genre`
- `tone`
- `rules jsonb`
- `regions jsonb`
- `music_mood`
- `visual_style`
- `created_by_agent_run_id`
- `approved_at`
- `created_at`

## 3. Character, scene, prop

### characters

Stable identity independent from generated versions.

Important columns:

- `id`
- `project_id`
- `code`: Cxx
- `name`
- `role_type`: `main | supporting | background`
- `description`
- `personality`
- `motivation`
- `relationships jsonb`
- `created_at`
- `updated_at`

Unique:

- `(project_id, code)`

### character_versions

Versioned character card and visual lock.

Important columns:

- `id`
- `character_id`
- `version_no`
- `status`: `candidate | locked | rejected | superseded`
- `profile jsonb`
- `visual_prompt`
- `negative_prompt`
- `full_body_asset_id`
- `turnaround_asset_id`
- `expression_pack_asset_id`
- `pose_pack_asset_id`
- `created_by_agent_run_id`
- `locked_by`
- `locked_at`
- `created_at`

Rule:

- exactly one locked version per character for active production.

### locations

Reusable world location.

Important columns:

- `id`
- `project_id`
- `name`
- `region`
- `description`
- `map_position jsonb`
- `created_at`
- `updated_at`

### scenes

Scene/set used by shots. A scene belongs to an episode but can reference a reusable location.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `location_id`
- `code`: Sxx
- `name`
- `time_of_day`
- `weather`
- `season`
- `atmosphere`
- `lighting`
- `color_palette`
- `continuity_notes`
- `created_at`
- `updated_at`

Unique:

- `(episode_id, code)`

### scene_versions

Scene concept version.

Important columns:

- `id`
- `scene_id`
- `version_no`
- `status`: `candidate | locked | rejected | superseded`
- `concept_prompt`
- `wide_establishing_asset_id`
- `background_plate_asset_id`
- `alternate_angle_asset_ids uuid[]`
- `created_by_agent_run_id`
- `locked_by`
- `locked_at`
- `created_at`

### props

Reusable prop identity.

Important columns:

- `id`
- `project_id`
- `code`: Pxx
- `name`
- `owner_character_id`
- `description`
- `plot_function`
- `continuity_notes`
- `created_at`
- `updated_at`

Unique:

- `(project_id, code)`

### prop_versions

Versioned prop reference.

Important columns:

- `id`
- `prop_id`
- `version_no`
- `status`
- `visual_prompt`
- `reference_asset_id`
- `created_by_agent_run_id`
- `locked_by`
- `locked_at`
- `created_at`

## 4. Shot and storyboard

### shots

Canonical shot card.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `scene_id`
- `sequence_no`
- `status`: `draft | ready | generating | review_needed | approved | rejected`
- `duration_ms`
- `start_time_ms`
- `end_time_ms`
- `dialogue`
- `narration`
- `action`
- `visual_description`
- `shot_size`
- `camera_angle`
- `camera_movement`
- `lens_focal_length`
- `camera_equipment`
- `lens_style`
- `aperture_style`
- `director_intent`
- `emotional_tone`
- `narrative_purpose`
- `created_by_agent_run_id`
- `created_at`
- `updated_at`

Indexes:

- `(episode_id, sequence_no)`
- `(scene_id)`

### shot_characters

Shot to locked character versions.

Columns:

- `shot_id`
- `character_id`
- `character_version_id`
- `role_in_shot`

### shot_props

Shot to locked prop versions.

Columns:

- `shot_id`
- `prop_id`
- `prop_version_id`

### storyboard_panels

Generated storyboard images/panels.

Important columns:

- `id`
- `shot_id`
- `panel_index`
- `grid_page`
- `grid_type`: `6 | 9 | 16 | 25`
- `image_asset_id`
- `source_asset_id`
- `status`
- `created_at`

### shot_keyframes

Start/end/reference frames for video.

Important columns:

- `id`
- `shot_id`
- `kind`: `start | end | reference | tail`
- `asset_id`
- `prompt`
- `status`
- `generation_job_id`
- `created_at`

### video_prompt_packs

Provider-specific prompt package for one shot or shot group.

Important columns:

- `id`
- `shot_id`
- `provider_id`
- `model_id`
- `prompt`
- `negative_prompt`
- `time_slices jsonb`
- `reference_asset_ids uuid[]`
- `params jsonb`
- `prompt_template_id`
- `created_by_agent_run_id`
- `created_at`

## 5. Asset and lineage

### assets

Generic media asset.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `asset_type`: `image | video | audio | subtitle | document | json`
- `purpose`: `character_ref | scene_concept | prop_ref | storyboard | keyframe | video_clip | voice | export | source`
- `storage_uri`
- `thumbnail_uri`
- `mime_type`
- `size_bytes`
- `duration_ms`
- `width`
- `height`
- `hash`
- `metadata jsonb`
- `created_by_user_id`
- `created_by_agent_run_id`
- `created_by_generation_job_id`
- `created_at`

Indexes:

- `(project_id, purpose)`
- `(episode_id, purpose)`
- `(hash)`

### artifact_edges

Lineage graph between assets/artifacts.

Important columns:

- `id`
- `project_id`
- `source_type`: `asset | story_analysis | character_version | scene_version | prop_version | shot | prompt_pack`
- `source_id`
- `target_type`
- `target_id`
- `relationship`: `derived_from | used_as_reference | generated_by | supersedes`
- `created_at`

Rule:

- every generated media must be reproducible from lineage + prompt + provider + params.

## 6. Workflow and agents

### workflow_templates

Versioned production SOP.

Important columns:

- `id`
- `name`
- `version`
- `graph jsonb`
- `is_default`
- `created_at`

### workflow_runs

One execution of a production SOP.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `workflow_template_id`
- `status`: `draft | running | waiting_approval | succeeded | failed | canceled`
- `current_node_key`
- `started_by`
- `started_at`
- `finished_at`
- `created_at`

### workflow_node_runs

Durable node execution.

Important columns:

- `id`
- `workflow_run_id`
- `node_key`
- `node_type`
- `status`
- `input_artifact_ids uuid[]`
- `output_artifact_ids uuid[]`
- `attempt_count`
- `error_code`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`

### agents

Agent definition.

Important columns:

- `id`
- `name`
- `role`
- `version`
- `system_prompt_template_id`
- `default_model_id`
- `enabled`
- `created_at`

### agent_runs

One agent invocation.

Important columns:

- `id`
- `workflow_node_run_id`
- `agent_id`
- `status`
- `input jsonb`
- `output jsonb`
- `model_provider_id`
- `model_id`
- `prompt_template_id`
- `prompt_render_id`
- `cost_cents`
- `latency_ms`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`

### approval_gates

Human-in-the-loop checkpoint.

Important columns:

- `id`
- `workflow_run_id`
- `workflow_node_run_id`
- `gate_type`: `story | character | scene | prop | storyboard | final_timeline`
- `status`: `pending | approved | rejected | changes_requested`
- `subject_type`
- `subject_id`
- `requested_by_agent_run_id`
- `reviewed_by`
- `review_note`
- `created_at`
- `reviewed_at`

## 7. Model providers and generation jobs

### model_providers

Provider account/config metadata. Secrets should be stored encrypted or in secret manager, not plain DB.

Important columns:

- `id`
- `organization_id`
- `provider_code`: `openai | gemini | kling | runway | luma | veo | comfyui | custom`
- `display_name`
- `base_url`
- `secret_ref`
- `enabled`
- `created_at`
- `updated_at`

### model_catalog

Specific model and capability metadata.

Important columns:

- `id`
- `provider_id`
- `model_code`
- `display_name`
- `model_type`: `llm | image | video | audio | lip_sync | embedding`
- `supported_tasks text[]`: `t2v | i2v | flf2v | continuation | tts | stt | prompt_extend`
- `supported_aspect_ratios text[]`
- `supported_durations_sec int[]`
- `max_reference_images`
- `max_reference_videos`
- `max_prompt_chars`
- `preferred_prompt_language`
- `supports_cancel`
- `license_policy`
- `cost_policy jsonb`
- `enabled`
- `created_at`

### prompt_templates

Versioned prompt templates.

Important columns:

- `id`
- `name`
- `purpose`
- `version`
- `template_text`
- `schema jsonb`
- `created_at`

### prompt_renders

Rendered prompt trace.

Important columns:

- `id`
- `prompt_template_id`
- `input jsonb`
- `rendered_text`
- `created_by_agent_run_id`
- `created_at`

### generation_jobs

Long-running provider call.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `workflow_node_run_id`
- `agent_run_id`
- `provider_id`
- `model_id`
- `task_type`: `llm | image | video | audio | lip_sync | export`
- `status`: `queued | running | succeeded | failed | canceled | timed_out`
- `provider_task_id`
- `request_key`
- `prompt_render_id`
- `input_asset_ids uuid[]`
- `output_asset_ids uuid[]`
- `params jsonb`
- `progress`
- `status_message`
- `cost_cents`
- `attempt_count`
- `error_code`
- `error_message`
- `blocked_reason`
- `queued_at`
- `started_at`
- `finished_at`
- `updated_at`
- `created_at`

Indexes:

- `(status, queued_at)`
- `(project_id, created_at)`
- `(provider_task_id)`
- `(request_key)`

### generation_job_events

Append-only status and provider event history.

Important columns:

- `id`
- `generation_job_id`
- `old_status`
- `new_status`
- `event_type`
- `provider_status`
- `message`
- `metadata jsonb`
- `created_at`

### job_attempts

One worker/provider attempt for a generation job.

Important columns:

- `id`
- `generation_job_id`
- `attempt_no`
- `worker_id`
- `status`
- `started_at`
- `finished_at`
- `error_code`
- `error_message`
- `provider_status`
- `metadata jsonb`

## 8. Timeline and export

### timelines

Episode timeline.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `version_no`
- `status`: `draft | approved | exported`
- `duration_ms`
- `created_by_agent_run_id`
- `created_at`
- `updated_at`

### timeline_tracks

Track layer.

Important columns:

- `id`
- `timeline_id`
- `track_type`: `video | image | audio | subtitle | effect`
- `name`
- `sequence_no`
- `muted`
- `locked`

### timeline_clips

Clip on track.

Important columns:

- `id`
- `track_id`
- `asset_id`
- `shot_id`
- `start_ms`
- `duration_ms`
- `source_start_ms`
- `source_duration_ms`
- `transition_in jsonb`
- `transition_out jsonb`
- `effects jsonb`
- `metadata jsonb`

### subtitle_segments

Subtitles can also be represented as clips, but a typed table helps editing/search.

Important columns:

- `id`
- `episode_id`
- `shot_id`
- `asset_id`
- `speaker_character_id`
- `text`
- `start_ms`
- `end_ms`
- `style jsonb`

### exports

Rendered outputs.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `timeline_id`
- `status`: `queued | rendering | succeeded | failed | canceled`
- `format`: `mp4 | webm | zip | pdf`
- `resolution`
- `fps`
- `output_asset_id`
- `generation_job_id`
- `error_message`
- `created_at`
- `finished_at`

## 9. Review, safety, continuity

### review_issues

Continuity and safety findings.

Important columns:

- `id`
- `project_id`
- `episode_id`
- `subject_type`: `shot | asset | prompt_pack | timeline | export`
- `subject_id`
- `issue_type`: `character_drift | scene_mismatch | prop_missing | dialogue_conflict | safety | copyright | quality`
- `severity`: `low | medium | high | blocking`
- `message`
- `suggested_fix`
- `created_by_agent_run_id`
- `status`: `open | acknowledged | fixed | ignored`
- `created_at`
- `resolved_at`

### cost_ledger

Cost accounting.

Important columns:

- `id`
- `organization_id`
- `project_id`
- `episode_id`
- `generation_job_id`
- `agent_run_id`
- `provider_id`
- `model_id`
- `amount_cents`
- `currency`
- `reason`
- `created_at`

### cost_budgets

Budget policy for organization/project/episode/workflow scopes.

Important columns:

- `id`
- `organization_id`
- `project_id`
- `episode_id`
- `workflow_run_id`
- `scope`: `organization | project | episode | workflow_run`
- `limit_cents`
- `warning_threshold_cents`
- `currency`
- `period_start`
- `period_end`
- `status`: `active | paused | exhausted | closed`
- `created_at`
- `updated_at`

### cost_reservations

Pre-call spending hold for expensive model jobs.

Important columns:

- `id`
- `budget_id`
- `generation_job_id`
- `workflow_run_id`
- `amount_cents`
- `status`: `reserved | committed | released | expired`
- `created_at`
- `updated_at`

### audit_events

Important user/system actions.

Important columns:

- `id`
- `organization_id`
- `project_id`
- `actor_user_id`
- `actor_agent_run_id`
- `action`
- `subject_type`
- `subject_id`
- `metadata jsonb`
- `created_at`

## Critical relationships

```text
Project
  -> Episodes
  -> StorySources / StoryAnalyses / WorldBible
  -> Characters -> CharacterVersions -> Assets
  -> Locations -> Scenes -> SceneVersions -> Assets
  -> Props -> PropVersions -> Assets
  -> Shots -> ShotCharacters / ShotProps / StoryboardPanels / Keyframes / PromptPacks
  -> WorkflowRuns -> WorkflowNodeRuns -> AgentRuns -> GenerationJobs -> Assets
  -> Timelines -> Tracks -> Clips -> Exports
  -> ReviewIssues / CostLedger / AuditEvents
```

## MVP table shortlist

If implementation must be constrained, start with:

1. users, organizations, organization_members
2. projects, episodes, story_sources, story_analyses
3. characters, character_versions
4. locations, scenes, scene_versions
5. props, prop_versions
6. shots, shot_characters, shot_props, shot_keyframes
7. assets, artifact_edges
8. model_providers, model_catalog, prompt_templates, prompt_renders, generation_jobs
9. workflow_templates, workflow_runs, workflow_node_runs, agents, agent_runs, approval_gates
10. timelines, timeline_tracks, timeline_clips, exports
11. review_issues, cost_ledger

## Go implementation notes

- Default API stack: Chi router, pgx + sqlc, golang-migrate, River/PostgreSQL jobs, OpenAPI YAML contracts, SSE for realtime status.
- Use domain structs with typed enums; avoid spreading `map[string]any` through business logic.
- Use `json.RawMessage` or typed structs for flexible JSONB fields at repository boundaries.
- Keep provider credentials out of logs and out of plain DB columns.
- Use transactions when locking versions:
  - reject/supersede old candidates,
  - mark selected version locked,
  - write audit event,
  - release approval gate.
- Use optimistic concurrency for timeline edits:
  - `version_no`,
  - `updated_at`,
  - or a separate revision table if collaborative editing is added later.
- Do not store large base64 media in PostgreSQL; store media in object storage and keep metadata/URI in `assets`.
- Treat workflow/job state in PostgreSQL as the product source of truth; queue state is only an execution transport.
- Prefer transactional enqueue when creating jobs together with domain rows.

## Workflow/job state machine reference

See `workflow-job-state-machine.md` for detailed workflow, node, agent run, generation job, retry, cancellation, budget reservation, and realtime event rules.

## API scaffold reference

See `go-api-scaffold-plan.md` for route groups, package boundaries, middleware, DTO/OpenAPI strategy, worker job kinds, and implementation order.

## Open questions for next step

- Go router default: Chi unless the team explicitly chooses Gin/Fiber.
- DB access default: pgx + sqlc unless the team explicitly chooses Ent/GORM.
- Queue default: River/PostgreSQL-first unless operations prefer Asynq/Redis.
- Multi-tenant boundary: organization-only or project-level collaborators?
