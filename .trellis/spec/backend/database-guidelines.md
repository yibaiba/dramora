# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

Use PostgreSQL as the backend source of truth. Planned access stack is pgx + sqlc with golang-migrate migrations.

---

## Query Patterns

- Store SQL migrations under `db/migrations`.
- Store future sqlc query files under `db/queries`.
- Repository interfaces and transaction helpers live under `internal/repo`.
- Services own transaction boundaries; repositories own SQL mapping.

---

## Migrations

Migration filenames should be ordered and descriptive:

```text
000001_create_identity.up.sql
000001_create_identity.down.sql
000002_create_projects.up.sql
000002_create_projects.down.sql
```

Migration order should follow the implementation plan:

1. users, organizations, organization_members
2. projects, episodes
3. assets and artifact lineage
4. workflow/generation job tables
5. timeline/export tables
6. generated story analysis artifacts

### Scenario: Production core persistence

#### 1. Scope / Trigger

- Trigger: project/episode CRUD and production read APIs introduced PostgreSQL migrations and pgx repositories.

#### 2. Signatures

Migration files:

```text
db/migrations/000001_create_identity.up.sql
db/migrations/000002_create_projects.up.sql
db/migrations/000003_create_production_core.up.sql
db/migrations/000004_create_story_analyses.up.sql
db/migrations/000005_create_story_maps_and_shots.up.sql
db/migrations/000006_create_shot_prompt_packs.up.sql
db/migrations/000008_add_exports_status_index.up.sql
```

Repository constructors:

```go
func OpenPostgres(ctx context.Context, databaseURL string) (*repo.DB, error)
func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository
func NewPostgresProductionRepository(pool *pgxpool.Pool) *PostgresProductionRepository
```

#### 3. Contracts

- `MANMU_DATABASE_URL` set: API uses PostgreSQL repositories.
- `MANMU_DATABASE_URL` empty: API uses in-memory repositories for smoke tests and local UI development only.
- Default organization id must exist in PostgreSQL after migrations:
  `00000000-0000-0000-0000-000000000001`.
- `db/queries/*.sql` is the sqlc source of truth even when hand-written pgx repositories exist.
- Media payloads must be object URIs in `assets.uri`, never base64 blobs.
- MVP asset candidate locking uses `assets.status = 'ready'`; draft candidates remain `assets.status = 'draft'`.
- Human approval gates are stored in `approval_gates` as first-class blockers before expensive generation/export phases.
- SD2/Seedance prompt packs are stored in `shot_prompt_packs` as the source-of-truth prompt artifact before video generation jobs are submitted.
- `generation_jobs.prompt`, `generation_jobs.params`, `generation_jobs.provider_task_id`, and `generation_jobs.result_asset_id` must be loaded by worker-facing repository reads so provider execution can resume from durable state.
- Nullable UUIDs exposed to API read models are normalized to empty string until typed nullable DTOs are introduced.
- Generated flexible story analysis output uses JSONB seed arrays first; promote to normalized character/scene/prop tables in later slices.
- Story source input is stored in `story_sources`; `story_analyses.story_source_id`, `outline`, and `agent_outputs` link deterministic or provider-backed analysis output to the source text.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| PostgreSQL pool parse/open/ping fails | container creation fails and API exits non-zero. |
| Project not found | repository returns `domain.ErrNotFound`; HTTP maps to 404. |
| Empty project name or episode title | service returns `domain.ErrInvalidInput`; HTTP maps to 400. |
| Duplicate episode number | repository returns `domain.ErrInvalidInput`; HTTP maps to 400. |
| Missing generation job/workflow/timeline row | repository returns `domain.ErrNotFound`; HTTP maps to 404. |
| Starting story analysis | create `workflow_runs`, `generation_jobs`, and `generation_job_events` in one DB transaction. |
| Saving episode timeline | upsert `timelines` by `episode_id`, set status to `saved`, and increment version on updates. |
| Saving invalid timeline graph | reject blank track kind/name, blank clip kind, negative timings, or clips that exceed timeline duration with `domain.ErrInvalidInput`; HTTP maps to 400. |
| Worker advances generation job | update `generation_jobs.status` and insert a matching `generation_job_events` row in one DB transaction. |
| Worker submits a Seedance video job | read the persisted job prompt/params, advance `queued -> submitting -> submitted`, call the Seedance adapter outside HTTP handlers, and persist `provider_task_id`. |
| Worker polls a Seedance video job | process `submitted`/`polling` jobs by `provider_task_id`; running provider tasks stay/advance to `polling`, failed tasks advance to `failed`, and completed tasks advance through `downloading -> postprocessing -> succeeded`. |
| Worker downloads a Seedance result | require a provider result URI, create a ready `video` asset, and persist `generation_jobs.result_asset_id` before advancing from `downloading` to `postprocessing`. |
| Worker advances queued export | move `exports.status` from `queued` to `rendering` to `succeeded` through `ProductionService.ProcessQueuedExports`; do not mark exports succeeded in the HTTP handler. |
| Worker finds a rendering export | resume it and advance to `succeeded` so a previous partial worker failure does not leave exports permanently stuck. |
| Story analysis job succeeds in no-op worker | update the job to `succeeded`, insert the event, and create one linked `story_analyses` artifact in one repository transaction. |
| Story analysis job has a saved story source | use the latest episode story source when generating summary, C/S/P seeds, outline, and agent outputs. |
| Story analysis job has no story source | use the local default source only as a development fallback; do not pretend user-provided source exists. |
| Story maps are seeded | create or update episode-scoped C/S/P rows from latest story analysis seeds. |
| Storyboard shots are seeded | create or update episode-scoped shot cards from scene maps and latest story analysis. |
| Asset candidates are seeded | create idempotent episode-scoped assets from C/S/P map rows with object-style URIs. |
| Asset candidate is locked | update the asset status to `ready` and return the updated asset. |
| Approval gates are seeded | create idempotent episode-scoped gates for existing story, C/S/P map, storyboard, and timeline artifacts. |
| Shot prompt pack is generated | upsert one `shot_prompt_packs` row by `(shot_id, preset)` with JSONB time slices, reference bindings, and provider params. |
| Timeline graph is saved | upsert timeline metadata, then replace tracks/clips in a repository transaction. |

#### 5. Good/Base/Bad Cases

- Good: service validates request intent, repository maps PostgreSQL errors to domain errors, handler maps domain errors to API envelope.
- Base: in-memory repositories are acceptable for route tests and smoke tests.
- Bad: route handler executes SQL directly or exposes raw `pgconn.PgError` to clients.

#### 6. Tests Required

- Handler tests for project/episode creation, listing, and validation.
- Service tests for required fields and not-found behavior.
- PostgreSQL integration tests should be added before relying on complex JSONB/transaction behavior.
- Production read routes should test empty list and 404 behavior until create/start endpoints exist.
- Timeline save routes should test save-then-read behavior through the HTTP layer.
- Timeline save routes should test invalid graph timing through the HTTP layer.
- Worker execution should be service-tested with the memory repo and later integration-tested against PostgreSQL before real providers run.
- Seedance worker tests should cover default fake mode without real `ARK_API_KEY`, persisted provider task ids, and polling completion.
- Seedance result tests should assert completed video jobs create a ready result asset and expose its id through the generation job read model.
- Export worker execution should be service-tested with the memory repo and route-tested by starting an export, processing queued exports, then reading the export status.
- Export worker tests should include resuming an export that is already in `rendering`.
- Story analysis artifact read routes should test generated artifact list/detail behavior through the HTTP layer.
- Story source routes should test save/list behavior and analysis output should assert `story_source_id`, non-empty `outline`, and multi-agent `agent_outputs`.
- Core map/asset/storyboard/timeline/export routes should test the end-to-end seed/lock/save/start route chain through the HTTP layer.
- Approval gate routes should test seed, approve, and request-changes behavior through the HTTP layer.
- Prompt pack routes should test SD2 preset, image-to-video task type, and `@image2` reference binding when multiple assets are locked.

### Scenario: Story source and multi-agent analysis artifacts

#### 1. Scope / Trigger

- Trigger: Studio accepts novel/story text and story-analysis workers generate inspectable outline/person/scene/prop output.
- Applies when changing story source routes, `story_analyses` JSONB output fields, deterministic analyzer behavior, OpenAPI schemas, or Studio hooks.

#### 2. Signatures

Database migration:

```text
db/migrations/000009_add_story_sources_and_analysis_outputs.up.sql
```

Frontend-facing API:

```text
GET  /api/v1/episodes/{episodeId}/story-sources
POST /api/v1/episodes/{episodeId}/story-sources
GET  /api/v1/episodes/{episodeId}/story-analyses
GET  /api/v1/story-analyses/{analysisId}
```

Repository/service signatures:

```go
ProductionRepository.CreateStorySource(ctx, params) (domain.StorySource, error)
ProductionRepository.LatestStorySource(ctx, episodeID) (domain.StorySource, error)
ProductionService.CreateStorySource(ctx, episode, input) (domain.StorySource, error)
```

#### 3. Contracts

- `CreateStorySourceRequest` fields:
  - `content_text` string, required, non-blank, max 20000 runes.
  - `source_type` optional enum-like string; unsupported values normalize to `novel`.
  - `title` optional string.
  - `language` optional string, defaults to `zh-CN`.
- `StoryAnalysis` response includes:
  - `story_source_id` string, empty only for development fallback/default source.
  - `outline` JSON array of `{ code, title, summary, visual_goal }`.
  - `agent_outputs` JSON array of `{ role, status, output, highlights }`.
- Local deterministic analysis may generate staged outputs without provider secrets, but downstream C/S/P seeds must still be derived from the selected source text.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Blank `content_text` | Return `domain.ErrInvalidInput`; HTTP maps to `400 invalid_request`. |
| `content_text` exceeds 20000 runes | Return `domain.ErrInvalidInput`; do not persist partial source. |
| Unknown episode id | Repository/service returns not found; HTTP maps to `404 not_found`. |
| No story source at worker completion | Use explicit default local source and leave `story_source_id` empty. |
| JSONB output cannot encode/decode | Return the real error; do not write success-shaped partial analysis. |

#### 5. Good/Base/Bad Cases

- Good: user saves novel text, starts story analysis, worker writes an analysis linked by `story_source_id` with non-empty `outline` and `agent_outputs`.
- Base: local dev without saved text still produces deterministic default output so Studio can test the flow.
- Bad: hardcoding fixed seeds for every story or keeping novel text only in React local state.

#### 6. Tests Required

- HTTP route test saves a story source, lists sources, starts analysis, processes queued jobs, and asserts linked source id plus non-empty outline/agent outputs.
- Go tests/build must pass after migration, repo, service, and DTO changes.
- Studio lint/build must pass after DTO/client/hook/UI changes.
- OpenAPI parse and GET/POST-only route scan must pass.

#### 7. Wrong vs Correct

#### Wrong

```go
Summary: "No-op story analyst extracted MVP seeds..."
CharacterSeeds: []string{"C01 protagonist"}
```

#### Correct

```go
source, err := s.production.LatestStorySource(ctx, generationJob.EpisodeID)
analysis := analyzeStorySource(source)
```

The correct form keeps source text as durable input and lets local deterministic analysis be replaced by provider-backed agents later without changing the API shape.

#### 7. Wrong vs Correct

##### Wrong

```go
project, err := db.QueryContext(ctx, "select ...")
```

inside an HTTP handler.

##### Correct

```go
project, err := api.projectService.GetProject(r.Context(), projectID)
```

Handlers call services; services call repositories; repositories own SQL.

---

### Scenario: Human approval gates

#### 1. Scope / Trigger

- Trigger: Manmu needs explicit human-in-the-loop blockers before storyboard/video/timeline/export phases.
- Applies when adding approval gate tables, route contracts, service review logic, Studio approval UI, or workflow gating behavior.

#### 2. Signatures

Database migration:

```text
db/migrations/000007_create_approval_gates.up.sql
```

Frontend-facing API:

```text
GET  /api/v1/episodes/{episodeId}/approval-gates
POST /api/v1/episodes/{episodeId}/approval-gates:seed
POST /api/v1/approval-gates/{gateId}:approve
POST /api/v1/approval-gates/{gateId}:request-changes
```

Repository/service signatures:

```go
ListApprovalGates(ctx context.Context, episodeID string) ([]domain.ApprovalGate, error)
SaveApprovalGate(ctx context.Context, params repo.SaveApprovalGateParams) (domain.ApprovalGate, error)
ReviewApprovalGate(ctx context.Context, params repo.ReviewApprovalGateParams) (domain.ApprovalGate, error)
SeedEpisodeApprovalGates(ctx context.Context, episode domain.Episode) ([]domain.ApprovalGate, error)
ApproveApprovalGate(ctx context.Context, gateID string, reviewedBy string, reviewNote string) (domain.ApprovalGate, error)
RequestApprovalChanges(ctx context.Context, gateID string, reviewedBy string, reviewNote string) (domain.ApprovalGate, error)
```

#### 3. Contracts

- `approval_gates` rows are durable blockers, not comments or UI-only state.
- One gate is unique by `(episode_id, gate_type, subject_type, subject_id)` so reseeding does not duplicate gates.
- MVP gate types are `story_direction`, `character_lock`, `scene_lock`, `prop_lock`, `storyboard_approval`, `final_timeline`, and `export_approval`.
- Gate statuses use `pending`, `approved`, `rejected`, `changes_requested`, and `canceled`.
- Review routes may accept optional `reviewed_by` and `review_note`; blank reviewer defaults to `studio`.
- Frontend state uses `['approval-gates', episodeId]`; do not copy approval rows into Zustand.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Missing/blank `episodeId` or `gateId` | Service returns `domain.ErrInvalidInput`; HTTP maps to 400 `invalid_request`. |
| Episode or gate does not exist | Repository/service returns `domain.ErrNotFound`; HTTP maps to 404 `not_found`. |
| Gate is reseeded for the same subject | Return the existing row; do not insert duplicates. |
| Pending gate is approved | Transition to `approved`, set `reviewed_by`, `review_note`, and `reviewed_at`. |
| Approved gate is reviewed again | Return `domain.ErrInvalidTransition`; HTTP maps to 409 `invalid_transition`. |

#### 5. Good/Base/Bad Cases

- Good: `POST /episodes/{episodeId}/approval-gates:seed` creates gates from existing story analysis, C/S/P map, storyboard, and timeline artifacts.
- Base: an episode with only story analysis creates a `story_direction` gate.
- Bad: the Studio marks a gate approved locally without persisting through `POST /approval-gates/{gateId}:approve`.

#### 6. Tests Required

- Domain status test must assert valid pending → approved and invalid terminal-state transitions.
- HTTP route test should cover artifact chain → seed gates → approve one gate → request changes on another.
- OpenAPI, `db/queries/*.sql`, Studio DTOs/client/hooks, and state-management spec must change in the same slice as route changes.
- Run the GET/POST-only route scan before finalizing because approval routes are frontend-facing actions.

#### 7. Wrong vs Correct

##### Wrong

```ts
setLocalGates((gates) => gates.map((gate) => ({ ...gate, status: 'approved' })))
```

##### Correct

```ts
approveApprovalGate(gateId, { reviewed_by: 'studio' })
```

Approval decisions are auditable backend state and must flow through service/repository persistence.

---

### Scenario: SD2 shot prompt pack persistence

#### 1. Scope / Trigger

- Trigger: storyboard shots need provider-specific SD2/Seedance prompt artifacts before video generation jobs are submitted.
- Applies when adding or changing prompt pack routes, DTOs, SQL, Studio hooks, or prompt rendering logic.

#### 2. Signatures

Database migration:

```text
db/migrations/000006_create_shot_prompt_packs.up.sql
```

Frontend-facing API:

```text
GET  /api/v1/storyboard-shots/{shotId}/prompt-pack
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:generate
POST /api/v1/storyboard-shots/{shotId}/videos:generate
```

Repository/service signatures:

```go
SaveShotPromptPack(ctx context.Context, params repo.SaveShotPromptPackParams) (domain.ShotPromptPack, error)
GetShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error)
GenerateShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error)
StartShotVideoGeneration(ctx context.Context, shotID string) (domain.GenerationJob, error)
```

#### 3. Contracts

- `shot_prompt_packs` is the source-of-truth artifact for model-ready prompt text and provider params.
- The unique key is `(shot_id, preset)` so regenerating `sd2_fast` updates the same shot/preset artifact.
- `time_slices`, `reference_bindings`, and `params` are JSONB; do not split these flexible provider payloads into columns until querying needs emerge.
- `reference_bindings` uses SD2-compatible tokens such as `@image1` and `@image2`; locked `assets.status = 'ready'` rows are eligible references.
- The first locked image reference may be role `first_frame`; additional image refs should be role `reference_image`.
- Prompt pack generation reads from existing shot and asset state; it must not synchronously call video generation providers from HTTP handlers.
- `POST /videos:generate` requires an existing prompt pack, creates or returns one queued `generation_jobs` row, and enqueues `jobs.JobKindGenerationSubmit` only when the row was newly created.
- Shot video job idempotency uses `request_key = "shot-video:{shotID}:{preset}"`; repeated requests for the same shot/preset must not create duplicate provider submissions.
- The queued job copies `provider`, `model`, `task_type`, `prompt`, `reference_bindings`, and `time_slices` from the prompt pack so the worker has a stable source-of-truth payload.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Missing/blank `shotId` | Service returns `domain.ErrInvalidInput`; HTTP maps to 400 `invalid_request`. |
| Shot does not exist | Repository returns `domain.ErrNotFound`; HTTP maps to 404 `not_found`. |
| Prompt pack has not been generated | `GET /prompt-pack` returns 404, not an empty fake success. |
| Video generation starts without a prompt pack | `POST /videos:generate` returns 404; do not auto-create a prompt pack in that handler. |
| Multiple locked assets exist | Prompt pack includes deterministic `@image1`, `@image2`, ... bindings up to provider max. |
| Prompt pack is regenerated | Upsert by `(shot_id, preset)` and update prompt JSONB/params without creating duplicates. |
| Video generation is requested repeatedly | Return the existing generation job for the stable `request_key` instead of inserting duplicates. |
| SQL JSON marshal/unmarshal fails | Return the error; do not silently drop prompt fields. |

#### 5. Good/Base/Bad Cases

- Good: `POST /storyboard-shots/{shotId}/prompt-pack:generate` stores an `sd2_fast` prompt pack with time slices, reference bindings, and provider params.
- Good: `POST /storyboard-shots/{shotId}/videos:generate` returns `202` with a queued `generation_job` derived from the stored prompt pack.
- Base: no locked assets still produces a text-to-video prompt pack with empty `reference_bindings`.
- Bad: handler constructs SQL, auto-generates missing prompt packs implicitly, or calls Ark video generation synchronously while serving prompt/video routes.

#### 6. Tests Required

- HTTP route test should cover the seed story analysis → story map → asset lock → storyboard → prompt pack chain.
- Assert generated prompt pack has `preset = "sd2_fast"`, `task_type = "image_to_video"` when references exist, and `@image2` when at least two assets are locked.
- Assert `POST /videos:generate` returns `202`, provider `seedance`, task type matching the prompt pack, and a queued generation job.
- Provider payload tests should cover request normalization separately in `internal/provider`; repository tests can remain memory-backed until PostgreSQL integration tests are introduced.
- OpenAPI, `db/queries/*.sql`, Studio DTOs/client/hooks, and README examples must change in the same slice as route changes.

#### 7. Wrong vs Correct

##### Wrong

```go
// Handler directly submits a long-running provider request.
task, err := seedanceClient.GenerateVideo(r.Context(), prompt)
```

##### Correct

```go
job, err := api.productionService.StartShotVideoGeneration(r.Context(), shotID)
writeJSON(w, http.StatusAccepted, Envelope{"generation_job": generationJobDTO(job)})
```

Prompt pack generation is a fast source-of-truth write; video generation routes only enqueue durable jobs, and actual provider submission belongs in asynchronous generation job execution.

---

## Naming Conventions

- Use plural snake_case table names.
- Use snake_case columns.
- Use `created_at` and `updated_at` timestamps for mutable tables.
- Use explicit status columns that map to domain status constants.
- Do not store media as base64 in PostgreSQL; store object/storage URIs.

---

## Common Mistakes

- Do not introduce SQLite-only behavior in tests for repository code; Manmu relies on PostgreSQL features such as JSONB, arrays, and transactions.
- Do not enqueue jobs outside the transaction that creates the source-of-truth row once River is introduced.
- Do not leave `.gitkeep` files in `db/migrations` or `db/queries` once real migration/query files exist; migration and sqlc tooling should only see relevant files.
- Do not create generation job rows without a stable `request_key`; it is the idempotency handle.
