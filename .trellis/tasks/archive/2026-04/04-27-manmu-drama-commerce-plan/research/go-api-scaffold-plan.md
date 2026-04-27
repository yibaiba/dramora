# Go API Route and Module Scaffold Plan

## Purpose

Define the first Go backend scaffold for Manmu: API route groups, package boundaries, middleware, DTO contracts, repositories, workers, and implementation order.

This is a planning document. It should guide the first code scaffold without forcing every domain table or feature to be implemented at once.

## Recommended Go stack

MVP defaults:

| Area | Recommendation | Why |
| --- | --- | --- |
| HTTP router | Chi | Lightweight, idiomatic, composable middleware, close to `net/http`. |
| DB driver | pgx v5 | PostgreSQL-first, fast, transaction-friendly, works with sqlc and River. |
| SQL access | sqlc | Type-safe SQL without hiding complex queries behind an ORM. |
| Migrations | golang-migrate | Simple CLI/library migration flow. |
| Queue | River + PostgreSQL | Transactional enqueue with domain writes; aligns with PostgreSQL source of truth. |
| Object storage | S3-compatible SDK abstraction | Portable across S3/OSS/COS/MinIO. |
| Logging | zerolog or slog | Structured logs; prefer stdlib `log/slog` if minimizing dependencies. |
| API contract | OpenAPI YAML + generated types later | Keeps React DTOs and Go handlers aligned. |
| Realtime | SSE first | One-way job/workflow progress is enough for MVP and simpler than WebSocket. |

Alternatives:

- Gin is acceptable if the team wants a more batteries-included web framework, but Chi keeps the API close to the standard library.
- Ent/GORM can speed up CRUD, but Manmu has many state-machine and transaction-heavy flows where explicit SQL is clearer.
- Asynq/Redis can replace River if operations prefer Redis-backed queues, but `generation_jobs` and workflow tables remain the source of truth.
- Temporal/Hatchet can be revisited after the MVP if workflow durability becomes more complex than River + explicit state tables.

## Repository layout

Recommended first scaffold:

```text
apps/
  api/
    main.go
  worker/
    main.go

cmd/
  migrate/
    main.go                # optional wrapper; CLI can also call golang-migrate directly

internal/
  app/
    config.go
    container.go
    shutdown.go

  httpapi/
    router.go
    middleware.go
    errors.go
    response.go
    sse.go
    dto/
    handlers/
      health.go
      auth.go
      projects.go
      episodes.go
      stories.go
      assets.go
      characters.go
      scenes.go
      props.go
      shots.go
      storyboard.go
      workflows.go
      agents.go
      jobs.go
      approvals.go
      timelines.go
      exports.go
      providers.go
      costs.go

  domain/
    identity.go
    project.go
    story.go
    character.go
    scene.go
    prop.go
    shot.go
    asset.go
    workflow.go
    generation.go
    timeline.go
    review.go
    cost.go
    errors.go

  service/
    project_service.go
    story_service.go
    asset_service.go
    workshop_service.go
    storyboard_service.go
    workflow_service.go
    generation_service.go
    timeline_service.go
    export_service.go
    provider_service.go
    cost_service.go

  repo/
    db.go
    tx.go
    queries/               # sqlc generated package or hand-written query boundary
    project_repo.go
    story_repo.go
    asset_repo.go
    workflow_repo.go
    generation_repo.go
    timeline_repo.go
    cost_repo.go

  workflow/
    template.go
    graph.go
    scheduler.go
    state_machine.go
    node_handlers.go
    approvals.go

  jobs/
    client.go
    workers.go
    types.go
    attempts.go
    reconciliation.go

  provider/
    adapter.go
    registry.go
    errors.go
    llm/
    image/
    video/
    audio/

  media/
    storage.go
    thumbnails.go
    ffmpeg.go
    signed_url.go

  cost/
    estimator.go
    budget.go
    reservation.go
    ledger.go

  realtime/
    events.go
    broker.go
    sse.go

api/
  openapi.yaml

db/
  migrations/
  queries/

configs/
  local.example.yaml
```

Boundary rules:

- `domain` contains typed entities, enums, and domain errors only.
- `httpapi` owns request parsing, auth context, DTO mapping, and HTTP errors.
- `service` owns use cases, transactions, permission checks, and orchestration calls.
- `repo` owns SQL and maps database rows to domain structs.
- `workflow` decides what node or gate should happen next.
- `jobs` owns River job args/workers and attempt lifecycle.
- `provider` owns external model API adapters.
- `media` owns object storage and export helpers.
- `cost` owns budgets, reservations, and ledger writes.
- `realtime` owns event fanout; it does not decide business state.

## Dependency direction

Keep dependencies one-way:

```text
apps/api
  -> internal/app
  -> internal/httpapi
  -> internal/service
  -> internal/domain + internal/repo + internal/workflow + internal/cost + internal/media

apps/worker
  -> internal/app
  -> internal/jobs
  -> internal/service + internal/provider + internal/media + internal/workflow + internal/cost

internal/repo
  -> internal/domain

internal/provider
  -> internal/domain
```

Forbidden:

- `repo` importing `httpapi`.
- `domain` importing `repo`, `httpapi`, or provider SDKs.
- React DTO naming leaking into database schema.
- Provider SDK response types leaking outside provider adapters.

## API route groups

Base prefix: `/api/v1`.

Frontend-facing API convention: use `GET` for reads and `POST` for all writes/actions. Do not expose `PATCH`, `PUT`, or `DELETE` between Studio and API; model partial updates, saves, deletes, cancellations, locks, and retries as explicit `POST` command routes.

### Health and meta

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/healthz` | Liveness. |
| GET | `/readyz` | DB/queue/storage readiness. |
| GET | `/meta/capabilities` | Frontend feature flags, supported task types, app version. |

### Identity and workspace

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/me` | Current user and memberships. |
| GET | `/organizations` | List user organizations. |
| GET | `/organizations/{orgId}` | Organization detail. |
| GET | `/organizations/{orgId}/members` | Membership list. |

MVP can use a simple auth stub if real auth is not implemented yet, but the route shape should already carry `organization_id` in access checks.

### Projects and episodes

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/projects` | Project list for current organization/user. |
| POST | `/projects` | Create project from idea/import metadata. |
| GET | `/projects/{projectId}` | Project overview. |
| POST | `/projects/{projectId}:update` | Update title/style/budget. |
| POST | `/projects/{projectId}:archive` | Archive project. |
| GET | `/projects/{projectId}/episodes` | List episodes. |
| POST | `/projects/{projectId}/episodes` | Create episode. |
| GET | `/episodes/{episodeId}` | Episode command center summary. |
| POST | `/episodes/{episodeId}:update` | Update episode metadata. |

### Story and script

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/episodes/{episodeId}/story-sources` | Add idea/text/file reference. |
| GET | `/episodes/{episodeId}/story-sources` | List story inputs. |
| POST | `/episodes/{episodeId}/story-analysis:run` | Start Story Analyst workflow node. |
| GET | `/episodes/{episodeId}/story-analyses` | List analysis versions. |
| GET | `/story-analyses/{analysisId}` | Analysis detail. |
| POST | `/story-analyses/{analysisId}:approve` | Approve story direction gate. |
| POST | `/story-analyses/{analysisId}:request-changes` | Request revision. |

### Character, scene, and prop workshops

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/episodes/{episodeId}/characters` | Character map. |
| POST | `/episodes/{episodeId}/characters` | Create character card. |
| POST | `/characters/{characterId}:update` | Update character. |
| POST | `/characters/{characterId}/versions:generate` | Generate reference candidates. |
| POST | `/character-versions/{versionId}:lock` | Lock accepted character version. |
| GET | `/episodes/{episodeId}/locations` | Location/scene map root. |
| GET | `/episodes/{episodeId}/scenes` | Scene list. |
| POST | `/episodes/{episodeId}/scenes` | Create scene card. |
| POST | `/scenes/{sceneId}/versions:generate` | Generate scene concept candidates. |
| POST | `/scene-versions/{versionId}:lock` | Lock scene version. |
| GET | `/episodes/{episodeId}/props` | Prop list. |
| POST | `/episodes/{episodeId}/props` | Create prop card. |
| POST | `/props/{propId}/versions:generate` | Generate prop candidates. |
| POST | `/prop-versions/{versionId}:lock` | Lock prop version. |

### Storyboard and shots

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/episodes/{episodeId}/storyboard` | Storyboard Kanban payload. |
| POST | `/episodes/{episodeId}/storyboard:generate` | Generate shot list from script/assets. |
| GET | `/shots/{shotId}` | Shot detail. |
| POST | `/shots/{shotId}:update` | Edit shot card fields. |
| POST | `/shots/{shotId}/prompt-pack:generate` | Generate provider-specific prompt pack. |
| POST | `/shots/{shotId}/keyframes:generate` | Generate keyframe candidates. |
| POST | `/shots/{shotId}/videos:generate` | Generate video variants. |
| POST | `/shots/{shotId}:approve` | Approve generated shot output. |
| POST | `/shots/{shotId}:send-to-timeline` | Create/update timeline clips from approved shot. |

### Assets and lineage

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/episodes/{episodeId}/assets` | Filtered asset library. |
| GET | `/assets/{assetId}` | Asset detail. |
| POST | `/assets/{assetId}:update` | Update metadata/tags. |
| GET | `/assets/{assetId}/lineage` | Source and target graph. |
| POST | `/assets/{assetId}:lock` | Lock an asset version/reference. |
| POST | `/assets/{assetId}:signed-url` | Create safe preview/download URL. |

### Workflow, agents, approvals

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/episodes/{episodeId}/workflows` | Create workflow run from default SOP. |
| GET | `/workflow-runs/{workflowRunId}` | Workflow detail. |
| POST | `/workflow-runs/{workflowRunId}:start` | Start/resume workflow. |
| POST | `/workflow-runs/{workflowRunId}:cancel` | Cancel workflow. |
| GET | `/workflow-runs/{workflowRunId}/nodes` | Node run list. |
| GET | `/workflow-runs/{workflowRunId}/agent-runs` | Agent activity feed. |
| GET | `/workflow-runs/{workflowRunId}/approvals` | Pending/completed gates. |
| POST | `/approval-gates/{gateId}:approve` | Approve gate. |
| POST | `/approval-gates/{gateId}:reject` | Reject gate. |
| POST | `/approval-gates/{gateId}:request-changes` | Request changes. |

### Story analysis artifacts

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/episodes/{episodeId}/story-analyses` | List generated story analysis artifacts for an episode. |
| GET | `/story-analyses/{analysisId}` | Story analysis artifact detail for C/S/P seed review. |

### Generation jobs

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/generation-jobs` | Global/episode filtered job list. |
| GET | `/generation-jobs/{jobId}` | Job detail. |
| GET | `/generation-jobs/{jobId}/attempts` | Attempt history. |
| POST | `/generation-jobs/{jobId}:retry` | Retry as a new candidate or attempt. |
| POST | `/generation-jobs/{jobId}:cancel` | Cancel locally and provider-side when possible. |

### Timeline and export

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/episodes/{episodeId}/timeline` | Current timeline. |
| POST | `/episodes/{episodeId}/timeline` | Save full timeline with optimistic version. |
| POST | `/timelines/{timelineId}/clips` | Add clip. |
| POST | `/timeline-clips/{clipId}:update` | Edit clip placement/effects. |
| POST | `/timeline-clips/{clipId}:remove` | Remove clip. |
| POST | `/timelines/{timelineId}:duplicate-version` | Create editable version. |
| POST | `/timelines/{timelineId}/exports` | Start export job. |
| GET | `/exports/{exportId}` | Export status/detail. |
| POST | `/exports/{exportId}:cancel` | Cancel export. |

### Providers, models, and costs

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/providers` | Enabled providers. |
| POST | `/providers` | Add provider config with secret reference. |
| POST | `/providers/{providerId}:update` | Update provider metadata. |
| GET | `/models` | Model catalog with capabilities. |
| GET | `/cost/budgets` | Budget list. |
| POST | `/cost/budgets` | Create/update project/episode/workflow budget. |
| GET | `/cost/ledger` | Cost ledger filtered by project/episode/job. |
| POST | `/cost/estimate` | Estimate cost before expensive generation. |

### Realtime

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/events/stream` | SSE stream scoped by project/episode/workflow. |

SSE query params:

- `project_id`
- `episode_id`
- `workflow_run_id`
- `last_event_id`

## Handler/service/repo pattern

Request flow:

```text
HTTP request
  -> middleware: request id, auth, org/project access, logging
  -> handler: parse + validate DTO
  -> service: use case + transaction + domain errors
  -> repo/provider/media/workflow/cost
  -> handler: map domain result to API DTO
```

Rules:

- Handlers do not build SQL queries.
- Repositories do not know HTTP or JSON response shapes.
- Services decide transactions and permission-sensitive operations.
- Long-running work returns quickly with `workflow_run_id`, `generation_job_id`, or `export_id`.
- Synchronous request timeout should be short; generation/export happens in workers.

## Error contract

Recommended error response:

```json
{
  "error": {
    "code": "provider_rate_limited",
    "message": "The selected video provider is rate limited. Try again later or switch provider.",
    "retryable": true,
    "details": {
      "provider_id": "uuid",
      "retry_after_sec": 60
    },
    "request_id": "req_..."
  }
}
```

Error mapping:

| Domain error | HTTP status |
| --- | --- |
| validation failed | 400 |
| unauthenticated | 401 |
| forbidden / wrong organization | 403 |
| not found | 404 |
| conflict / stale timeline version / locked asset | 409 |
| budget exceeded / approval required | 402 or 409; prefer 409 for product-gated workflow |
| provider rejected prompt | 422 |
| rate limited | 429 |
| internal / provider unknown | 500 / 502 / 503 |

Do not expose:

- provider credentials,
- raw secret refs,
- full provider request headers,
- private object storage URLs,
- unsafe prompt moderation internals beyond user-actionable messages.

## Middleware

MVP middleware stack:

- request ID,
- structured request logging,
- panic recovery,
- CORS for Studio domain,
- auth context,
- organization/project access resolver,
- request body size limits,
- rate limit hooks,
- response compression for JSON where useful.

For SSE:

- auth required,
- no response buffering,
- heartbeat events,
- Last-Event-ID resume support if event storage is available.

## DTO and OpenAPI strategy

Start with hand-written Go DTOs plus `api/openapi.yaml` as the contract source. Once route shapes stabilize, generate clients/types.

Options:

- `ogen`: strong OpenAPI-first server/client codegen.
- `oapi-codegen`: mature OpenAPI codegen ecosystem.

MVP recommendation:

1. Write `api/openapi.yaml` for stable DTOs and enums.
2. Keep handler implementation manual until routes settle.
3. Generate TypeScript client later or use OpenAPI fetch client.
4. Add request/response examples for React Studio pages.

Stable DTO principles:

- Use enum strings matching backend domain status.
- Use ISO-8601 timestamps.
- Use integer minor money units plus `currency`.
- Use `progress` as `0..100` or null.
- Return preview/download URLs separately from asset storage URI.

## Worker scaffold

`apps/worker` starts River and registers job kinds.

Initial job kinds:

```text
workflow.schedule
workflow.node.execute
agent.run
generation.submit
generation.poll
generation.download
media.postprocess
export.render
jobs.reconcile
cost.reservation_expire
```

Worker rules:

- Workers load job row by ID and re-check persisted status before side effects.
- Provider submit uses idempotency keys where available.
- Poll/download/postprocess can be retried independently.
- Worker completion and domain writes happen in transactions where possible.
- Every attempt writes `job_attempts` or `generation_job_events`.

## Configuration

`configs/local.example.yaml` should define:

```yaml
server:
  addr: ":8080"
database:
  url: "postgres://..."
storage:
  provider: "s3|minio|oss|cos"
  bucket: "manmu-local"
realtime:
  transport: "sse"
queue:
  driver: "river"
providers:
  openai:
    enabled: false
    secret_ref: "env:OPENAI_API_KEY"
  kling:
    enabled: false
    secret_ref: "env:KLING_API_KEY"
cost:
  default_currency: "USD"
```

Secrets:

- local dev may use env vars,
- production should use secret manager or encrypted secret refs,
- never store plaintext API keys in provider rows.

## First implementation slice

Recommended order:

1. Go module + `apps/api` health/readiness.
2. Config loading + structured logging + request IDs.
3. PostgreSQL connection + migrations folder.
4. Users/orgs/projects/episodes minimal tables and routes.
5. Asset + story source minimal tables/routes.
6. Model providers/model catalog routes.
7. Workflow run + generation job tables and read-only routes.
8. River setup + worker process + `workflow.schedule` no-op job.
9. Story analysis start route that creates workflow/job rows transactionally.
10. SSE stream for workflow/job events.
11. Storyboard/shot read/update routes.
12. Timeline read/save route with optimistic concurrency.
13. Export job route and worker stub.

This order gives React Studio stable backend contracts early without requiring real model providers on day one.

## API-to-UI mapping

| React surface | API groups |
| --- | --- |
| Project list | `/projects`, `/episodes` |
| Episode command center | `/episodes/{id}`, `/workflow-runs/{id}`, `/cost/*`, `/review-issues` later |
| Script/story analysis | `/story-sources`, `/story-analysis:*`, `/approval-gates` |
| Asset library | `/assets`, `/assets/{id}/lineage`, workshop routes |
| Character workshop | `/characters`, `/character-versions:*` |
| Scene/prop workshop | `/scenes`, `/scene-versions:*`, `/props`, `/prop-versions:*` |
| Storyboard Kanban | `/storyboard`, `/shots/*`, `/generation-jobs` |
| Agent Board | `/workflow-runs/*`, `/agent-runs`, `/approvals`, `/events/stream` |
| Jobs page | `/generation-jobs`, `/generation-jobs/{id}/attempts` |
| Timeline editor | `/timeline`, `/timeline-clips` |
| Export page | `/exports`, `/timelines/{id}/exports` |

## Testing plan for scaffold

When implementation begins:

- Unit test domain state transitions.
- Repository tests should run against PostgreSQL, not SQLite, because JSONB, arrays, and transactions matter.
- Handler tests should cover validation, auth/org access, and error mapping.
- Worker tests should verify idempotent retries and no duplicate cost commits.
- Contract tests should compare OpenAPI examples with handler responses.

## Open decisions

- Confirm Chi as default router.
- Confirm sqlc + pgx as default DB access.
- Confirm River/PostgreSQL as default queue.
- Confirm `log/slog` vs zerolog.
- Confirm whether API should be OpenAPI-first from day one or documented-first then generated later.
