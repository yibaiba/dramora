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
| `MANMU_WORKER_QUEUES` | No | `default` | Comma-separated worker queues. |

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Invalid duration env var | `LoadConfig` returns an error and app exits non-zero. |
| Empty CSV worker queue env | fallback to `default`. |
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

## Examples

- `apps/api/main.go`: thin process entrypoint.
- `internal/httpapi/router.go`: route composition.
- `internal/domain/status.go`: pure domain status contracts.
