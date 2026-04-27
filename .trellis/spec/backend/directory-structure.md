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
| `MANMU_DEFAULT_ORGANIZATION_ID` | No | `00000000-0000-0000-0000-000000000001` | Default org used before auth/multitenancy UI exists. |
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
