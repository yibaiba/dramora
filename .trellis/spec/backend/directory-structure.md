# Directory Structure

> How backend code is organized in this project.

---

## Overview

The backend is a Go modular monolith. It starts as one deployable API and one worker process, but code is split by responsibility so model providers, jobs, repositories, and media/export can evolve independently.

---

## Directory Layout

```text
apps/
├── api/                 # HTTP API entrypoint
└── worker/              # Background worker entrypoint

api/
└── openapi.yaml         # Public API contract stub

configs/
└── local.example.yaml   # Example local config

db/
├── migrations/          # PostgreSQL migrations
└── queries/             # Future sqlc queries

internal/
├── app/                 # Config, container, server lifecycle
├── cost/                # Budget/reservation contracts
├── domain/              # Pure domain enums, entities, validation
├── httpapi/             # Chi router, middleware, handlers, DTO helpers
├── jobs/                # Queue/job type and worker contracts
├── media/               # Storage/export boundaries
├── provider/            # LLM/image/video/audio provider adapters
├── realtime/            # Stable SSE/event contracts
├── repo/                # SQL repository and transaction boundaries
└── workflow/            # Typed workflow graph and orchestration rules
```

---

## Module Organization

### 1. Scope / Trigger

- Trigger: the first backend scaffold established command, API, environment, and package boundary contracts.
- Applies to all Go backend work under `apps/`, `internal/`, `api/`, `db/`, and `configs/`.

### 2. Signatures

- API process:
  ```bash
  go run ./apps/api
  ```
- Worker process:
  ```bash
  go run ./apps/worker
  ```
- Router constructor:
  ```go
  func NewRouter(cfg httpapi.RouterConfig) http.Handler
  ```
- Config loader:
  ```go
  func LoadConfig() (app.Config, error)
  ```

### 3. Contracts

- `apps/api` may import `internal/app` and `internal/httpapi`, but should not contain business logic.
- `apps/worker` may import `internal/app` and `internal/jobs`, but should not contain job logic.
- `internal/domain` must not import infrastructure packages such as HTTP, SQL, storage, queue, or provider SDKs.
- `internal/httpapi` owns HTTP routing and response shape only.
- `internal/service` will own use cases and transactions when introduced.
- `internal/repo` owns SQL mapping and transaction helpers.
- `internal/provider` hides external model SDKs behind interfaces.
- `internal/workflow` owns typed workflow graph rules and approval gates.

Environment keys:

| Key | Required | Default | Meaning |
| --- | --- | --- | --- |
| `MANMU_ENV` | No | `local` | Runtime environment label. |
| `MANMU_HTTP_ADDR` | No | `:8080` | API listen address. |
| `MANMU_READ_HEADER_TIMEOUT` | No | `5s` | HTTP read header timeout; duration or integer seconds. |
| `MANMU_SHUTDOWN_TIMEOUT` | No | `10s` | Graceful shutdown timeout; duration or integer seconds. |
| `MANMU_DATABASE_URL` | No | empty | PostgreSQL URL. Empty uses in-memory repositories for local smoke tests. |
| `MANMU_INLINE_WORKER` | No | `true` when `MANMU_ENV=local`, otherwise `false` | Run the worker loop inside the API process so local Studio actions auto-complete queued jobs. |
| `MANMU_WORKER_QUEUES` | No | `default` | Comma-separated worker queues. |

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Invalid duration env var | `LoadConfig` returns an error and app exits non-zero. |
| Empty CSV worker queue env | fallback to `default`. |
| Invalid `MANMU_INLINE_WORKER` boolean | `LoadConfig` returns an error and app exits non-zero. |
| Missing database URL | API uses in-memory repositories; do not claim PostgreSQL persistence in that mode. |
| Domain package needs HTTP/SQL/provider import | reject; move code to `httpapi`, `repo`, or `provider`. |
| New route lacks OpenAPI entry | update `api/openapi.yaml` in the same change. |

### 5. Good/Base/Bad Cases

- Good: add `internal/provider/foo.go` with an interface or adapter that hides a third-party API.
- Base: add placeholder handlers returning stable envelope shapes while repositories are not ready.
- Bad: put SQL calls or provider API calls directly in `apps/api/main.go` or a route handler.

### 6. Tests Required

- New domain state machine or validator: unit tests in `internal/domain`.
- New route: handler test covering status code and JSON response shape.
- New config key: test default, valid custom value, and invalid value if parsing is non-trivial.
- New package boundary: `go test ./...` and `go build ./...` must pass without import cycles.

### 7. Wrong vs Correct

#### Wrong

```go
// apps/api/main.go
db.QueryContext(ctx, "select * from projects")
```

#### Correct

```go
// apps/api/main.go
router := httpapi.NewRouter(httpapi.RouterConfig{Logger: logger})
```

Business logic belongs below `internal/service` and persistence belongs below `internal/repo`, not in entrypoints.

---

## Naming Conventions

- Package directories use lowercase names.
- Entrypoints live under `apps/<process>/main.go`.
- Handler files live in `internal/httpapi` and are grouped by route surface, for example `projects.go`, `workflows.go`, and `timelines.go`.
- Domain files name the concept they own, for example `status.go`, `transitions.go`, and `errors.go`.

---

## Scenario: Frontend-facing API uses GET/POST only

### 1. Scope / Trigger

- Trigger: Studio-to-API route contracts must stay simple and compatible with clients/proxies that only allow `GET` and `POST`.
- Applies to every route under `/api/v1` that is consumed by the React Studio.

### 2. Signatures

Allowed frontend-facing HTTP signatures:

```text
GET  /api/v1/<resource>
POST /api/v1/<resource>
POST /api/v1/<resource>:<action>
```

Current command-style examples:

```text
POST /api/v1/episodes/{episodeId}/story-analysis/start
POST /api/v1/episodes/{episodeId}/timeline
POST /api/v1/story-map-characters/{characterId}/character-bible:save
POST /api/v1/timeline-clips/{clipId}:remove
```

### 3. Contracts

- Use `GET` only for reads that do not change state.
- Use `POST` for creates, updates, saves, deletes/removes, locks, approvals, retries, cancellations, exports, and workflow actions.
- Do not add frontend-facing `PUT`, `PATCH`, or `DELETE` routes.
- Command-style `POST` routes may use either nested resources (`/episodes/{episodeId}/timeline`) or explicit actions (`/assets/{assetId}:lock`).
- `api/openapi.yaml`, HTTP handlers, route tests, and frontend client methods must use the same method.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| New frontend route needs mutation semantics | Implement as `POST`, not `PUT`/`PATCH`/`DELETE`. |
| Existing research map suggests non-GET/POST method | Convert to equivalent `POST ...:<action>` route before implementation. |
| OpenAPI method differs from Chi route | Treat as contract drift; fix before validation passes. |
| Frontend client uses non-GET/POST method | Treat as a spec violation; update client and tests. |

### 5. Good/Base/Bad Cases

- Good: `POST /api/v1/episodes/{episodeId}/timeline` saves a draft timeline.
- Base: `POST /api/v1/assets/{assetId}:lock` records a command without exposing REST verb nuance.
- Bad: `PATCH /api/v1/assets/{assetId}` or `DELETE /api/v1/timeline-clips/{clipId}` in frontend-facing contracts.

### 6. Tests Required

- Handler tests must use the same `GET`/`POST` method as OpenAPI.
- Frontend API client methods must build requests with only `GET` or `POST`.
- Run a search for `PUT`, `PATCH`, `DELETE`, `.Put(`, `.Patch(`, `.Delete(`, `http.MethodPut`, `http.MethodPatch`, and `http.MethodDelete` before finalizing route changes.

### 7. Wrong vs Correct

#### Wrong

```go
r.Put("/episodes/{episodeId}/timeline", api.saveEpisodeTimeline)
```

#### Correct

```go
r.Post("/episodes/{episodeId}/timeline", api.saveEpisodeTimeline)
```

The correct form preserves the project convention while still expressing a state-changing save command.

---

## Scenario: Storyboard workspace aggregate read contract

### 1. Scope / Trigger

- Trigger: Storyboard 成为独立页面后，需要一个 episode-scoped 聚合读模型来承接页面主数据，而不是把主读状态拆散到多个资源查询。
- Applies when changing storyboard workspace routes, OpenAPI schemas, HTTP handlers, service read models, or Studio hooks consuming the aggregate route.

### 2. Signatures

Frontend-facing API:

```text
GET /api/v1/episodes/{episodeId}/storyboard-workspace
```

Service signature:

```go
ProductionService.GetStoryboardWorkspace(ctx context.Context, episodeID string) (service.StoryboardWorkspace, error)
```

Handler/DTO boundary:

```go
Envelope{"storyboard_workspace": storyboardWorkspaceDTO(workspace)}
```

### 3. Contracts

- The route is episode-scoped and read-only; keep all storyboard writes on the existing resource/action POST routes.
- `GET /storyboard-workspace` returns `200` with a full aggregate envelope even when story map arrays, shots, assets, approval gates, or generation jobs are empty.
- The aggregate response should include:
  - `episode_id`
  - `summary`
  - `story_map`
  - `storyboard_shots`
  - `assets`
  - `approval_gates`
  - `generation_jobs`
- `storyboard_shots[]` may enrich each shot with `scene`, `prompt_pack`, and `latest_generation_job`.
- `prompt_pack` inside the aggregate payload is a summary/readiness projection, not a replacement for the full prompt-pack detail route.
- Do not overload `GET /episodes/{episodeId}/storyboard-shots` with aggregate workspace concerns; keep that route a pure shot list.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Blank `episodeId` | Return invalid input from service and JSON error from handler. |
| Episode has no seeded storyboard production data yet | Return `200` with empty aggregate collections instead of failing the whole route. |
| Story map not seeded yet | Return `200` with empty `story_map` arrays and `summary.story_map_ready = false`. |
| Shot has no generated prompt pack | Return `prompt_pack = null` for that shot; do not fail the whole workspace read. |
| Episode has no generation jobs | Return `generation_jobs: []`. |
| Existing resource routes still needed | Keep `GET /storyboard-shots`, `POST :update`, prompt-pack routes, and approval routes (`:approve`, `:request-changes`, `:resubmit`) unchanged. |

### 5. Good/Base/Bad Cases

- Good: `GET /api/v1/episodes/{episodeId}/storyboard-workspace` returns the page's main read model while writes stay resource-oriented.
- Base: workspace shot includes `scene` and prompt-pack summary when those artifacts exist, and `null` when they do not.
- Bad: adding a POST aggregate workspace endpoint for reads, or stuffing workspace summary fields into `GET /storyboard-shots`.

### 6. Tests Required

- HTTP route test must cover a seeded episode and assert workspace response includes summary, shot scene metadata, prompt-pack summary, and latest generation job projection.
- HTTP route test should cover the empty/not-yet-seeded episode case and assert `200` with empty aggregates instead of `404`.
- OpenAPI, handler DTOs, frontend DTOs/client/hooks, and README examples must change in the same slice as the route.
- Route convention scan must confirm no frontend-facing `PUT`, `PATCH`, or `DELETE`.

### 7. Wrong vs Correct

#### Wrong

```go
r.Get("/episodes/{episodeId}/storyboard-shots", api.getStoryboardWorkspace)
```

by repurposing the shot-list route for the workspace aggregate.

#### Correct

```go
r.Get("/episodes/{episodeId}/storyboard-workspace", api.getStoryboardWorkspace)
r.Get("/episodes/{episodeId}/storyboard-shots", api.listStoryboardShots)
```

The correct form keeps resource reads and workspace aggregate reads as separate contracts.

---

## Scenario: Character Bible save contract

### 1. Scope / Trigger

- Trigger: Studio needs to persist character-specific bible metadata from `AssetsGraphPage`.
- Applies when changing story-map character routes, OpenAPI schemas, handler DTOs, service validation, or repo persistence for `character_bible`.

### 2. Signatures

Frontend-facing API:

```text
POST /api/v1/story-map-characters/{characterId}/character-bible:save
```

Service signature:

```go
ProductionService.SaveCharacterBible(ctx context.Context, characterID string, input SaveCharacterBibleInput) (domain.Character, error)
```

Handler/DTO boundary:

```go
Envelope{"story_map_item": characterDTO(character)}
```

### 3. Contracts

- The route is character-scoped and mutation-only; do not overload `story-map:seed` for bible edits.
- Persist Character Bible only for story-map characters unless a new explicit contract is added for scenes/props.
- `anchor` is required and must be validated in the service layer.
- `character_bible.reference_assets[]` stores angle-to-asset bindings as `{ angle, asset_id }` inside the existing JSON payload.
- Successful saves must round-trip through existing story-map projections, so `GET /episodes/{episodeId}/story-map` and `GET /episodes/{episodeId}/storyboard-workspace` both expose `character_bible`.
- Store normalized/trimmed bible fields; do not preserve empty-string noise or duplicate list entries.
- Reference assets must belong to the same episode, be `assets.kind = "character"`, match the character node code via `assets.purpose`, and already be in `ready` status before they can be persisted.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Blank `characterId` | Return invalid input via existing JSON error helpers. |
| Blank bible anchor | Return invalid input; do not accept an empty persisted bible. |
| Reference asset angle not enabled in `reference_angles` | Return invalid input instead of saving a dangling binding. |
| Reference asset is not `ready` or does not belong to the character node | Return invalid input; do not persist the mapping. |
| Unknown character id | Return the repo/service not-found error through the handler. |
| Save succeeds | Return the updated `story_map_item` envelope and keep read projections consistent. |

### 5. Good/Base/Bad Cases

- Good: `POST /api/v1/story-map-characters/{characterId}/character-bible:save` updates one character bible, including `reference_assets`, and subsequent `GET /story-map` shows the saved payload.
- Base: optional palette/expressions/reference angles may be omitted while the anchor remains required.
- Bad: adding a `PATCH /story-map-characters/{characterId}` route or hiding persistence only inside the frontend.

### 6. Tests Required

- HTTP route test must reject unlocked/mismatched reference assets, then save Character Bible and read back the character from both `story-map` and `storyboard-workspace`.
- Repo coverage must ensure sqlite/postgres/in-memory implementations all preserve `character_bible`.
- OpenAPI and frontend client/type updates must land in the same slice as the handler route.

---

## Scenario: Storyboard edit and prompt save command contracts

### 1. Scope / Trigger

- Trigger: Studio lets editors persist storyboard shot card changes and edited SD2 direct prompts.
- Applies when changing storyboard shot routes, prompt pack routes, OpenAPI schemas, service methods, or Studio API hooks.

### 2. Signatures

Frontend-facing API:

```text
POST /api/v1/storyboard-shots/{shotId}:update
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save
```

Service signatures:

```go
ProductionService.UpdateStoryboardShot(ctx, shotID, UpdateStoryboardShotInput) (domain.StoryboardShot, error)
ProductionService.SaveShotPromptPack(ctx, shotID, SaveShotPromptPackInput) (domain.ShotPromptPack, error)
```

### 3. Contracts

- `UpdateStoryboardShotRequest` fields:
  - `title` string, required, non-blank.
  - `description` string, optional.
  - `prompt` string, required, non-blank.
  - `duration_ms` integer, required, positive.
- `SaveShotPromptPackRequest` fields:
  - `direct_prompt` string, required, non-blank.
- `POST ...:update` returns `{ "storyboard_shot": StoryboardShot }`.
- `POST .../prompt-pack:save` returns `{ "prompt_pack": ShotPromptPack }`.
- Prompt-pack save must preserve provider/model/preset/task type/negative prompt/time slices/reference bindings/params when a pack already exists. If no pack exists, build the default pack from the shot before replacing `direct_prompt`.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Blank `shotId` | Return invalid input from service and JSON error from handler. |
| Blank shot `title` or `prompt` | Return invalid input; do not persist partial shot edits. |
| Non-positive `duration_ms` | Return invalid input. |
| Blank `direct_prompt` | Return invalid input; do not create an empty prompt pack. |
| Unknown shot id | Return not found. |
| OpenAPI/client/hook route mismatch | Treat as contract drift; update all layers before validation. |

### 5. Good/Base/Bad Cases

- Good: editor saves title, description, prompt, and duration through `POST /api/v1/storyboard-shots/{shotId}:update`; the response updates `['storyboard-shots', episodeId]`.
- Base: editor saves only a changed `direct_prompt`; the backend returns the full prompt pack with preserved model metadata.
- Bad: adding `PATCH /api/v1/storyboard-shots/{shotId}` or saving edited prompts only in React local state while showing success.

### 6. Tests Required

- HTTP route test must update a seeded shot and assert changed title/description/prompt/duration in the response.
- HTTP route test must save an edited prompt pack and assert `direct_prompt` changed.
- Frontend validation must run `npm run lint` and `npm run build` after adding DTO/client/hook types.
- Route convention scan must confirm no frontend-facing `PUT`, `PATCH`, or `DELETE`.

### 7. Wrong vs Correct

#### Wrong

```ts
await fetch(`/api/v1/storyboard-shots/${shotId}`, {
  method: 'PATCH',
  body: JSON.stringify({ title }),
})
```

#### Correct

```ts
useUpdateStoryboardShot().mutate({
  shotId,
  request: { title, description, prompt, duration_ms },
})
```

The correct form keeps route strings in `src/api/client.ts`, server state behind `src/api/hooks.ts`, and mutation semantics on command-style `POST` routes.

---

## Scenario: One-click episode production seed command

### 1. Scope / Trigger

- Trigger: Studio needs to continue from a completed story analysis into story map, candidate assets, storyboard cards, and human gates with one command.
- Applies when changing production pipeline routes, aggregate service orchestration, OpenAPI schemas, route tests, or Studio API hooks.

### 2. Signatures

Frontend-facing API:

```text
POST /api/v1/episodes/{episodeId}/production:seed
```

Service signature:

```go
ProductionService.SeedEpisodeProduction(ctx, episode) (SeedEpisodeProductionResult, error)
```

### 3. Contracts

- The command requires a completed story analysis for the episode; it must not silently create fake analysis data.
- The service composes existing production use cases in order: story map, episode assets, storyboard shots, approval gates.
- The response envelope returns `{ "story_map": StoryMap, "assets": Asset[], "storyboard_shots": StoryboardShot[], "approval_gates": ApprovalGate[] }`.
- Studio client code must expose the route through `src/api/client.ts` and `src/api/hooks.ts`, invalidating all affected query keys after success.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Missing episode id or unknown episode | Return JSON not-found/invalid error through existing handler helpers. |
| No completed story analysis | Return not found from the story-map seed step; do not fabricate seeds. |
| Partial step failure | Return the real error; do not report success-shaped partial output. |
| OpenAPI/client/hook route mismatch | Treat as contract drift; update all layers before validation. |

### 5. Good/Base/Bad Cases

- Good: after story-analysis worker completion, Studio calls `POST /episodes/{episodeId}/production:seed` and refreshes story map, assets, storyboard shots, and approval gates.
- Base: editors may still use the individual seed commands for step-by-step debugging.
- Bad: adding a frontend-only "success" state without backend artifacts, or using `PATCH`/`PUT` for the aggregate command.

### 6. Tests Required

- HTTP route test must complete a story-analysis job, call `production:seed`, and assert non-empty story map, assets, storyboard shots, and approval gates.
- Frontend validation must run `npm run lint` and `npm run build` after adding DTO/client/hook types.
- Route convention scan must confirm no frontend-facing `PUT`, `PATCH`, or `DELETE`.

---

## Examples

- `apps/api/main.go`: thin process entrypoint.
- `internal/httpapi/router.go`: route composition.
- `internal/domain/status.go`: pure domain status contracts.

---

## Scenario: Hybrid provider adapter boundary

### 1. Scope / Trigger

- Trigger: external model providers need testable adapters that can run without credentials locally but use real APIs when runtime secrets are configured.
- Applies to provider code under `internal/provider` and service/job orchestration that prepares provider requests.

### 2. Signatures

Provider adapter construction:

```go
func NewSeedanceAdapterFromEnv() *provider.SeedanceAdapter
func NewSeedanceAdapter(apiKey string, baseURL string, client *http.Client) *provider.SeedanceAdapter
func (a *SeedanceAdapter) SubmitGeneration(ctx context.Context, input SeedanceRequestInput) (SeedanceGenerationTask, error)
```

Runtime environment:

```text
ARK_API_KEY          optional; empty means fake mode
ARK_API_BASE_URL     optional; defaults to the Ark generation task URL
```

### 3. Contracts

- `internal/provider` owns external HTTP payloads and provider-specific response decoding.
- Domain structs must not import provider SDKs or HTTP types.
- Empty `ARK_API_KEY` means fake mode and must remain usable in tests/local dev.
- Non-empty `ARK_API_KEY` enables Ark POST submission mode; never commit real keys or examples that look like real keys.
- Provider errors must be returned explicitly and must not include secret values in client-facing messages.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| `ARK_API_KEY` empty | Adapter returns fake queued task metadata without network I/O. |
| `ARK_API_KEY` set | Adapter sends `POST` to configured Ark base URL with bearer auth. |
| Ark response status is non-2xx | Return an explicit provider error; do not mark generation succeeded. |
| Ark response lacks task id | Return an error instead of fabricating a real task id. |
| Provider request needs image references | Normalize them to content entries and reference tokens inside `internal/provider`. |

### 5. Good/Base/Bad Cases

- Good: unit tests use `httptest.Server` to assert bearer auth and payload shape without real Ark credentials.
- Base: fake mode returns deterministic task metadata for local smoke tests.
- Bad: route handler imports a provider SDK or reads `ARK_API_KEY` directly.

### 6. Tests Required

- Adapter request builder tests should assert model id, task type, duration defaults, and `@image2` reference preservation.
- Hybrid mode tests should assert fake mode without key and Ark POST mode with an injected HTTP client/server.
- Go validation must pass with `GOTOOLCHAIN=local`; do not add dependencies that upgrade Go or Chi incidentally.

### 7. Wrong vs Correct

#### Wrong

```go
func (api *api) generateVideo(w http.ResponseWriter, r *http.Request) {
    key := os.Getenv("ARK_API_KEY")
    // build provider HTTP request in the handler
}
```

#### Correct

```go
adapter := provider.NewSeedanceAdapterFromEnv()
task, err := adapter.SubmitGeneration(ctx, input)
```

Provider integration stays behind `internal/provider`; handlers and domain code only see project-owned service/domain contracts.

---

## Scenario: SQLite default persistence with PostgreSQL fallback

### 1. Scope / Trigger

- Trigger: the project needs zero-config local persistence without requiring a PostgreSQL instance.
- Applies when changing database initialization, repository wiring, or container startup.

### 2. Signatures

SQLite open:

```go
func OpenSQLite(ctx context.Context, dbPath string) (*SQLiteDB, error)
```

Container fallback:

```go
if cfg.DatabaseURL != "" {
    // PostgreSQL path
} else {
    // SQLite path: .data/data.db
}
```

### 3. Contracts

- `MANMU_DATABASE_URL` set: API uses PostgreSQL repositories (unchanged).
- `MANMU_DATABASE_URL` empty: API uses SQLite repositories with auto-migration at `.data/data.db`.
- `MANMU_DATA_DIR` overrides the SQLite data directory (default: `.data`).
- SQLite opens with WAL mode, foreign keys enabled, busy timeout 5s.
- SQLite migrations run on every startup via `CREATE TABLE IF NOT EXISTS`.
- `.data/` is in `.gitignore`.
- SQLite queries use `?` placeholders, `TEXT` for UUIDs, `TEXT` for JSON columns, `strftime(...)` for timestamps.
- PostgreSQL queries remain unchanged with `$N::uuid`, `::jsonb`, `now()`.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| `MANMU_DATA_DIR` parent missing | `OpenSQLite` creates the directory via `os.MkdirAll`. |
| SQLite migration DDL fails | Container creation fails and API exits non-zero. |
| Foreign key violation in SQLite | Detected via string match `"FOREIGN KEY constraint failed"`, mapped to `domain.ErrNotFound`. |
| Unique violation in SQLite | Detected via string match `"UNIQUE constraint failed"`, mapped to `domain.ErrInvalidInput`. |
| SQLite `INSERT` without `RETURNING` | Do `INSERT` then `SELECT` by ID for read-back. |
| SQLite `UPDATE` without `RETURNING` | Do `UPDATE`, check `RowsAffected`, then `SELECT` by ID. |

### 5. Good/Base/Bad Cases

- Good: `DATABASE_URL` empty, API starts with SQLite, all routes work identically to PostgreSQL.
- Base: developer explicitly sets `MANMU_DATA_DIR=./test-data` for a test environment.
- Bad: importing `modernc.org/sqlite` from `internal/domain` or handlers.

### 6. Tests Required

- Existing handler tests pass without modification (they use memory repos).
- `go build ./...` and `go test ./...` pass with `GOTOOLCHAIN=local`.

### 7. Wrong vs Correct

#### Wrong

```go
// Handler directly opens SQLite
db, _ := sql.Open("sqlite", "./data.db")
```

#### Correct

```go
// Container opens SQLite; handlers use repo interfaces
sqliteDB, err := repo.OpenSQLite(ctx, dbPath)
projectRepo = repo.NewSQLiteProjectRepository(sqliteDB.DB)
```

---

## Scenario: Admin provider config management

### 1. Scope / Trigger

- Trigger: administrators configure AI capability endpoints (chat/image/video/audio) via Studio management UI.
- Applies when changing provider config routes, DTOs, repository, or Studio admin pages.

### 2. Signatures

Frontend-facing API:

```text
GET  /api/v1/admin/providers
POST /api/v1/admin/providers:save
POST /api/v1/admin/providers/{capability}:test
```

Service signatures:

```go
ProviderService.ListProviderConfigs(ctx) ([]domain.ProviderConfig, error)
ProviderService.SaveProviderConfig(ctx, SaveProviderConfigInput) (domain.ProviderConfig, error)
ProviderService.TestProviderConfig(ctx, capability) TestProviderResult
```

### 3. Contracts

- `GET /admin/providers` returns `{ "providers": [...] }` with `api_key` masked (first 4 + last 4 chars).
- `POST /admin/providers:save` requires `capability` (chat|image|video|audio), `base_url`, `api_key`, `model`.
- `POST /admin/providers/{capability}:test` returns `200` with `{ "test_result": { ok, model, latency_ms, error } }` always — test failure is not an HTTP error.
- Provider configs are stored in `provider_configs` table (SQLite/PostgreSQL), upserted by `capability` unique key.
- `ProviderService` is `nil` when using memory repos (no SQLite); admin routes return 500 gracefully.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Missing capability/base_url/api_key/model | Return `domain.ErrInvalidInput`; HTTP maps to 400. |
| Invalid capability (not chat/image/video/audio) | Return `domain.ErrInvalidInput`; HTTP maps to 400. |
| Test endpoint unreachable | Return `{ ok: false, error: "连接失败: ..." }` with HTTP 200. |
| Test endpoint returns 401 | Return `{ ok: false, error: "API Key 无效 (401)" }` with HTTP 200. |

### 5. Good/Base/Bad Cases

- Good: admin saves chat endpoint, tests connection, sees green "连接成功" with latency.
- Base: no endpoints configured; agents fall back to deterministic analysis.
- Bad: returning raw provider errors to frontend; storing API keys in plaintext logs.

---

## Scenario: DAG workflow engine and agent execution

### 1. Scope / Trigger

- Trigger: story analysis transitions from deterministic to LLM-driven multi-agent execution.
- Applies when changing workflow engine, agent service, or story analysis completion logic.

### 2. Signatures

Engine:

```go
workflow.NewEngine(graph *Graph, bb *Blackboard, executor NodeExecutor) *Engine
engine.EnableCheckpointing(workflowID string, store CheckpointStore)
engine.Resume(checkpoint *Checkpoint) error
engine.Execute(ctx context.Context) error
engine.Runs() map[string]*NodeRun
```

Agent service:

```go
AgentService.MakeNodeExecutor(sourceText string) workflow.NodeExecutor
AgentService.IsAvailable(ctx context.Context) bool
ProductionService.GetWorkflowRunDetail(ctx context.Context, id string) (WorkflowRunDetail, error)
ProductionRepository.SaveWorkflowCheckpoint(ctx, workflowRunID, payload) error
ProductionRepository.LoadWorkflowCheckpoint(ctx, workflowRunID) ([]byte, error)
```

### 3. Contracts

- Phase 1 graph: `story_analyst → outline_planner → character_analyst | scene_analyst | prop_analyst` (last 3 parallel).
- Engine executes nodes in topological order; sibling nodes with all dependencies satisfied run concurrently.
- Failed node causes dependent downstream nodes to be `skipped`, not `failed`.
- Blackboard is the inter-agent communication channel; agents write by role, downstream agents read upstream output.
- Checkpoint snapshots capture both node-run state and Blackboard state; resume restores Blackboard first, then restores node runs.
- Resume re-queues interrupted `running` nodes as `waiting`; `succeeded`, `failed`, and `skipped` remain terminal.
- Enabled checkpointing must emit monotonic sequence snapshots so concurrent saves do not regress to older progress.
- Story-analysis checkpoint persistence is stored against the existing `workflow_runs` row and reloaded by `workflow_run_id`; do not create a second ad-hoc checkpoint table for this path.
- Worker recovery must continue `story_analysis` jobs from `postprocessing`, because that is the status where `completeGeneratedStoryAnalysis` executes the DAG and can now resume from a saved checkpoint.
- `GET /api/v1/workflow-runs/{workflowRunId}` should expose checkpoint observability as a read-model summary (sequence, saved time, node counts, blackboard roles) instead of dumping raw checkpoint payloads.
- `AgentService.IsAvailable` checks whether `chat` provider config exists; if not, `completeGeneratedStoryAnalysis` falls back to deterministic `analyzeStorySource`.
- LLM agents use OpenAI-compatible `/chat/completions` endpoint configured via admin provider management.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Chat endpoint not configured | `IsAvailable` returns false; deterministic fallback runs. |
| LLM returns non-JSON | Agent highlights fall back to truncated raw output. |
| DAG has cycle | `Engine.Execute` returns error before running any node. |
| Resume checkpoint contains `running` nodes | Treat them as interrupted work, reset to `waiting`, and continue from restored Blackboard state. |
| Checkpoint store receives out-of-order saves | Keep the newest sequence; do not overwrite newer progress with stale snapshots. |
| Story-analysis job is already in `postprocessing` | Worker resumes `completeGeneratedStoryAnalysis` instead of ignoring the job as already processed. |
| Node executor panics | Recovered by `middleware.Recoverer`; node marked `failed`. |

### 5. Good/Base/Bad Cases

- Good: admin configures chat endpoint → story analysis runs 5 LLM agents via DAG → agent outputs with highlights displayed in AgentBoard.
- Good: workflow resumes from a saved checkpoint and only reruns unfinished nodes.
- Good: a restarted worker picks up a `postprocessing` story-analysis job, reloads the saved checkpoint from `workflow_runs`, and completes the remaining nodes.
- Base: no chat endpoint → deterministic fallback, zero behavior change for existing users.
- Bad: calling LLM synchronously in HTTP handler; storing LLM raw responses in domain entities.
