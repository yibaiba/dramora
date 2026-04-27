# Manmu AI Manju Studio Implementation Plan

## Goal

Start implementing Manmu as an AI-generated manju studio with:

- Go backend control plane,
- React Studio frontend,
- PostgreSQL source of truth,
- external model API first,
- durable workflow/job/cost tracking,
- narrow internal timeline editor,
- server-side MP4 export.

## Product backbone

Implementation should follow this combined reference model:

```text
AIComicBuilder-like staged pipeline
  + AIYOU-like typed workflow graph internally
  + Seedance2-like C/S/P asset and time-coded prompt discipline
  + Openjourney-like candidate selection UX
  + Go/PostgreSQL durable workflow/job/cost backend
  + React Studio for approval, generation control, and final editing
```

## Locked MVP decisions

- Backend: Go modular monolith.
- Router: Chi.
- DB: PostgreSQL.
- DB access: pgx + sqlc.
- Migrations: golang-migrate.
- Queue: River + PostgreSQL transactional jobs.
- Realtime: SSE first.
- Storage: S3-compatible abstraction; local dev can use filesystem/MinIO.
- Frontend: React + TypeScript.
- Studio state: TanStack Query for server state; Zustand for high-frequency editor state.
- Timeline: build a narrow internal timeline editor.
- Export: Go/River/FFmpeg worker first.
- Model strategy: external API first; no default self-hosted GPU in MVP.
- Workflow UI: fixed SOP + Agent Board; no freeform workflow builder in v1.

## First implementation slice

The first code slice should prove the foundation, not the full product:

1. Create Go module and backend scaffold.
2. Add `apps/api` with health/readiness routes.
3. Add config loading and structured logging.
4. Add PostgreSQL connection placeholder and migrations folder.
5. Add `internal/domain` typed enums/entities for project, episode, workflow, generation job, asset, timeline.
6. Add `internal/httpapi` router, error response shape, and response helpers.
7. Add `internal/repo` interface boundaries and transaction helper skeleton.
8. Add `apps/worker` with River-compatible job type scaffolding, even if River dependency is initially behind an interface.
9. Add OpenAPI contract stub for health, project, workflow, generation job, timeline, export, SSE.
10. Add README update with local dev commands.

Exit criteria for slice 1:

- `go test ./...` passes.
- `go test` covers domain status transition basics.
- `go build ./...` passes.
- Health route compiles and can be run locally.
- No provider secrets or real model keys are required.

Slice 1 status: complete.

Implemented:

- Go module pinned to Go 1.21-compatible dependencies.
- `apps/api` with health/readiness, capability, placeholder project/episode/workflow/job/timeline, and SSE routes.
- `apps/worker` scaffold.
- `internal/app`, `internal/httpapi`, `internal/domain`, `internal/repo`, `internal/workflow`, `internal/jobs`, `internal/provider`, `internal/media`, `internal/cost`, and `internal/realtime` package boundaries.
- Domain status enums and transition tests for workflow runs and generation jobs.
- OpenAPI stub, migration/query folders, and local config example.
- README backend instructions.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- API smoke checks for `/healthz`, `/api/v1/meta/capabilities`, and `/api/v1/events/stream`.

Next implementation slice:

- Add River-backed enqueue interface after generation job rows exist.
- Add create/start endpoints for story analysis workflow runs.
- Add timeline save/update backed by PostgreSQL.
- Add React Studio scaffold against the stabilized project/episode/job/timeline contracts.

Slice 2 status: complete.

Implemented:

- PostgreSQL migrations for identity, projects, episodes, assets, artifact lineage, workflow runs, workflow node runs, generation jobs/events, timelines/tracks/clips, and exports.
- sqlc query source files for project/episode and production read models.
- pgxpool database wiring with in-memory fallback when `MANMU_DATABASE_URL` is empty.
- Project/episode domain models, repository interfaces, PostgreSQL repository, memory repository, and service layer.
- Real project and episode CRUD handlers with JSON validation, stable DTOs, and error mapping.
- Read-only workflow run, generation job, and timeline services/routes replacing placeholder responses.
- Handler and service tests for CRUD and production read route behavior.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- API smoke for project creation, episode creation/listing, generation job list, and workflow 404 behavior.

## Backend package plan

```text
apps/api
apps/worker
internal/app
internal/httpapi
internal/domain
internal/service
internal/repo
internal/workflow
internal/jobs
internal/provider
internal/media
internal/cost
internal/realtime
api/openapi.yaml
db/migrations
db/queries
configs/local.example.yaml
```

Rules:

- `domain` imports no infrastructure packages.
- `httpapi` owns HTTP DTOs and maps domain errors to HTTP responses.
- `service` owns use cases and transaction boundaries.
- `repo` owns SQL/database mapping.
- `workflow` decides next nodes and approval gates.
- `jobs` moves durable work through queues.
- `provider` hides external model SDK details.
- `media` hides storage/export implementation.
- `cost` owns budget/reservation/ledger rules.
- `realtime` broadcasts stable events only.

## Minimal route scaffold

Start with:

```text
GET /healthz
GET /readyz
GET /api/v1/meta/capabilities
GET /api/v1/projects
POST /api/v1/projects
GET /api/v1/projects/{projectId}
GET /api/v1/projects/{projectId}/episodes
POST /api/v1/projects/{projectId}/episodes
GET /api/v1/episodes/{episodeId}
GET /api/v1/generation-jobs
GET /api/v1/generation-jobs/{jobId}
GET /api/v1/episodes/{episodeId}/story-analyses
GET /api/v1/story-analyses/{analysisId}
GET /api/v1/episodes/{episodeId}/timeline
GET /api/v1/workflow-runs/{workflowRunId}
POST /api/v1/episodes/{episodeId}/timeline
GET /api/v1/events/stream
```

Routes can return placeholder/in-memory responses in slice 1 if database integration is not yet ready, but DTOs and error shape should already match the intended contract.
Frontend-facing APIs use only `GET` for reads and `POST` for writes/actions.

## Domain status enums to implement first

- `ProjectStatus`
- `EpisodeStatus`
- `WorkflowRunStatus`
- `WorkflowNodeRunStatus`
- `AgentRunStatus`
- `GenerationJobStatus`
- `AssetStatus`
- `ApprovalGateStatus`
- `TimelineStatus`
- `ExportStatus`

Status transitions should be explicit helper methods or validators.

## Data model to prioritize in migrations

Migration order for real database integration:

1. users, organizations, organization_members
2. projects, episodes
3. story_sources, story_analyses
4. assets, artifact_edges
5. characters, character_versions
6. locations, scenes, scene_versions
7. props, prop_versions
8. shots, shot_characters, shot_props, shot_keyframes, prompt_packs
9. model_providers, model_catalog, prompt_templates, prompt_renders
10. workflow_templates, workflow_runs, workflow_node_runs, agents, agent_runs, approval_gates
11. generation_jobs, generation_job_events, job_attempts
12. cost_budgets, cost_reservations, cost_ledger
13. timelines, timeline_tracks, timeline_clips, subtitle_segments
14. exports, review_issues, audit_events

## Frontend scaffold after backend foundation

Start React only after Go foundation compiles:

```text
apps/studio
  Vite React + TypeScript
  Tailwind
  shadcn/ui/Radix
  TanStack Query
  Zustand
```

First Studio pages:

1. StudioShell.
2. Project list.
3. Episode command center.
4. Agent Board placeholder.
5. Storyboard Kanban placeholder.
6. Asset library placeholder.
7. Timeline editor placeholder.
8. Jobs bottom rail / SSE event viewer.

Frontend slice status: complete.

Implemented:

- `apps/studio` Vite React TypeScript application.
- Tailwind v4 Vite plugin, dark professional Studio styling, responsive shell, and reduced-motion handling.
- TanStack Query server state hooks for projects, episodes, and generation jobs.
- Zustand store for selected project and local event rail.
- Typed API DTO/client layer aligned with current OpenAPI route contracts.
- StudioShell with Project list, Episode command center, Agent Board, Storyboard Kanban, Asset library, Timeline placeholder, and Jobs rail.
- Vite dev proxy for backend `/api`, `/healthz`, and `/readyz`.

Verified:

- `cd apps/studio && npm run lint`
- `cd apps/studio && npm run build`
- Dev smoke through Vite proxy:
  - `GET /api/v1/projects`
  - `POST /api/v1/projects`

Workflow start slice status: complete.

Implemented:

- `POST /api/v1/episodes/{episodeId}/story-analysis/start`.
- Backend service creates a `workflow_run` and queued `generation_job`.
- PostgreSQL implementation writes workflow/job/event rows in one transaction.
- In-memory implementation supports local smoke tests and frontend dev.
- `jobs.Client` abstraction with no-op enqueue for the current slice.
- Studio Episode command center can start story analysis for an episode and refresh job data.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- `cd apps/studio && npm run lint`
- `cd apps/studio && npm run build`
- API smoke for create project, create episode, start story analysis, and list generation jobs.

Worker no-op execution slice status: complete.

Implemented:

- Repository methods and sqlc query sources for listing queued `generation_jobs` and advancing job status with `generation_job_events`.
- `ProductionService.ProcessQueuedGenerationJobs` no-op executor that moves queued jobs through the existing transition path to `succeeded`.
- `jobs.Executor`, `ExecutionSummary`, and `Worker.RunOnce` boundaries so River can replace polling later without changing service contracts.
- `apps/worker` now builds an app container and uses `ProductionService` instead of being a log-only scaffold.
- Worker and service tests for no-op generation job execution.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`

Timeline save slice status: complete.

Implemented:

- `POST /api/v1/episodes/{episodeId}/timeline` for saving an episode timeline draft.
- PostgreSQL timeline upsert keyed by `episode_id`, preserving stable timeline identity and incrementing version on subsequent saves.
- In-memory timeline save/read behavior for route tests, local Studio development, and smoke tests.
- Studio Episode command center "Save timeline" action wired through typed API client and TanStack mutation.
- OpenAPI and README route documentation for the episode-scoped timeline save contract.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- `cd apps/studio && npm run lint`
- `cd apps/studio && npm run build`
- API smoke for create project, create episode, save timeline, and read timeline back.

Agent Board data slice status: complete.

Implemented:

- Agent Board now derives SOP step state from real `generation_jobs` rows loaded through `GET /api/v1/generation-jobs`.
- Project-scoped job filtering keeps Agent Board aligned with the selected Studio project.
- Story Analyst reflects queued/running/succeeded/blocked job states; downstream SOP steps become ready after story analysis succeeds.
- Agent Board keeps text status labels and job counts so status is not color-only.
- Enforced the new frontend-facing API convention: reads use `GET`, writes/actions use `POST`; no implemented Studio/API route uses `PUT`, `PATCH`, or `DELETE`.

Verified:

- `cd apps/studio && npm run lint`
- `cd apps/studio && npm run build`

Core implementation map status:

1. Foundation: complete.
2. Project/episode CRUD: complete.
3. Workflow start + durable generation job row: complete.
4. Worker no-op execution over queued jobs: complete.
5. Timeline metadata save/load: complete.
6. Agent Board real job data: complete.
7. Story analysis artifacts persistence/read API: complete.
8. Character/scene/prop maps: complete.
9. Storyboard shot cards: complete.
10. Timeline tracks/clips save/load: complete.
11. Export job/render scaffold: complete.

Story analysis artifact slice status: complete.

Implemented:

- `story_analyses` migration with episode-scoped versions and generation job linkage.
- Domain/repository/service support for generated story analysis artifacts.
- Worker no-op execution now creates a structured story analysis artifact when a `story_analysis` job succeeds.
- Read APIs: `GET /api/v1/episodes/{episodeId}/story-analyses` and `GET /api/v1/story-analyses/{analysisId}`.
- OpenAPI, README, and sqlc query source updates for the new artifact contract.
- Studio typed client/hooks and Story Analysis panel showing latest summary, seed counts, and themes.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `cd apps/studio && npm run lint -- --quiet`
- `cd apps/studio && npm run build`

Studio core panels slice status: complete.

Implemented:

- C/S/P Asset Library panel backed by `useStoryMap` and `useSeedStoryMap`.
- Storyboard Kanban panel backed by `useStoryboardShots` and `useSeedStoryboardShots`.
- Timeline editor action that saves storyboard-derived tracks/clips through `POST /episodes/{episodeId}/timeline`.
- Export action that starts the episode export scaffold and displays queued export status.
- Added `GET /episodes/{episodeId}/timeline` Studio hook so timeline save/load round trips through TanStack Query.

Deferred:

- Freeform canvas, advanced trimming/splitting, and manual clip drag editing until asset candidate generation exists.

Remaining core modules slice status: complete.

Implemented:

- `characters`, `scenes`, `props`, and `storyboard_shots` migrations.
- Episode-scoped GET/POST seed APIs for C/S/P maps and storyboard shot cards.
- Timeline graph save/load support for tracks and clips on the existing `POST /episodes/{episodeId}/timeline` route.
- Export scaffold with `POST /episodes/{episodeId}/exports` and `GET /exports/{exportId}`.
- PostgreSQL and in-memory repository implementations for all four core modules.
- Typed Studio client/hooks for the new GET/POST-only API contracts.

Verified:

- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- `cd apps/studio && npm run lint -- --quiet`
- `cd apps/studio && npm run build`

## Validation plan

Slice 1:

- `go test ./...`
- `go build ./...`

After React scaffold:

- package manager install lockfile.
- frontend type-check/build script.
- basic lint if project scaffold includes lint.

## References

- `prd.md`
- `research/go-api-scaffold-plan.md`
- `research/go-backend-domain-model.md`
- `research/workflow-job-state-machine.md`
- `research/react-studio-ui-map.md`
- `research/timeline-editor-tech-selection.md`
- `research/ai-manju-github-deep-dive-implementation-map.md`
