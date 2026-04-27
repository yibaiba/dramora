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
- SD2/Seedance prompt packs are stored in `shot_prompt_packs` as the source-of-truth prompt artifact before video generation jobs are submitted.
- Nullable UUIDs exposed to API read models are normalized to empty string until typed nullable DTOs are introduced.
- Generated flexible story analysis output uses JSONB seed arrays first; promote to normalized character/scene/prop tables in later slices.

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
| Worker advances generation job | update `generation_jobs.status` and insert a matching `generation_job_events` row in one DB transaction. |
| Story analysis job succeeds in no-op worker | update the job to `succeeded`, insert the event, and create one linked `story_analyses` artifact in one repository transaction. |
| Story maps are seeded | create or update episode-scoped C/S/P rows from latest story analysis seeds. |
| Storyboard shots are seeded | create or update episode-scoped shot cards from scene maps and latest story analysis. |
| Asset candidates are seeded | create idempotent episode-scoped assets from C/S/P map rows with object-style URIs. |
| Asset candidate is locked | update the asset status to `ready` and return the updated asset. |
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
- Worker execution should be service-tested with the memory repo and later integration-tested against PostgreSQL before real providers run.
- Story analysis artifact read routes should test generated artifact list/detail behavior through the HTTP layer.
- Core map/asset/storyboard/timeline/export routes should test the end-to-end seed/lock/save/start route chain through the HTTP layer.
- Prompt pack routes should test SD2 preset, image-to-video task type, and `@image2` reference binding when multiple assets are locked.

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
```

Repository/service signatures:

```go
SaveShotPromptPack(ctx context.Context, params repo.SaveShotPromptPackParams) (domain.ShotPromptPack, error)
GetShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error)
GenerateShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error)
```

#### 3. Contracts

- `shot_prompt_packs` is the source-of-truth artifact for model-ready prompt text and provider params.
- The unique key is `(shot_id, preset)` so regenerating `sd2_fast` updates the same shot/preset artifact.
- `time_slices`, `reference_bindings`, and `params` are JSONB; do not split these flexible provider payloads into columns until querying needs emerge.
- `reference_bindings` uses SD2-compatible tokens such as `@image1` and `@image2`; locked `assets.status = 'ready'` rows are eligible references.
- The first locked image reference may be role `first_frame`; additional image refs should be role `reference_image`.
- Prompt pack generation reads from existing shot and asset state; it must not synchronously call video generation providers from HTTP handlers.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Missing/blank `shotId` | Service returns `domain.ErrInvalidInput`; HTTP maps to 400 `invalid_request`. |
| Shot does not exist | Repository returns `domain.ErrNotFound`; HTTP maps to 404 `not_found`. |
| Prompt pack has not been generated | `GET /prompt-pack` returns 404, not an empty fake success. |
| Multiple locked assets exist | Prompt pack includes deterministic `@image1`, `@image2`, ... bindings up to provider max. |
| Prompt pack is regenerated | Upsert by `(shot_id, preset)` and update prompt JSONB/params without creating duplicates. |
| SQL JSON marshal/unmarshal fails | Return the error; do not silently drop prompt fields. |

#### 5. Good/Base/Bad Cases

- Good: `POST /storyboard-shots/{shotId}/prompt-pack:generate` stores an `sd2_fast` prompt pack with time slices, reference bindings, and provider params.
- Base: no locked assets still produces a text-to-video prompt pack with empty `reference_bindings`.
- Bad: handler constructs SQL or calls Ark video generation synchronously while serving prompt-pack generation.

#### 6. Tests Required

- HTTP route test should cover the seed story analysis → story map → asset lock → storyboard → prompt pack chain.
- Assert generated prompt pack has `preset = "sd2_fast"`, `task_type = "image_to_video"` when references exist, and `@image2` when at least two assets are locked.
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
pack, err := api.productionService.GenerateShotPromptPack(r.Context(), shotID)
writeJSON(w, http.StatusCreated, Envelope{"prompt_pack": shotPromptPackDTO(pack)})
```

Prompt pack generation is a fast source-of-truth write; actual video submission belongs in asynchronous generation job execution.

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
