# Workflow, Job State Machine, and Cost Control

## Purpose

Define the first executable orchestration model for Manmu's AI manju production backend.

This document turns the earlier domain model into concrete runtime rules for:

- fixed multi-agent production SOP,
- durable workflow and node execution,
- long-running provider jobs,
- retry/cancel/polling behavior,
- budget reservation and cost ledger,
- frontend status contracts for Agent Board, Storyboard Kanban, task queue, and export.

## Open-source references

| Reference | What to borrow | What to avoid in MVP |
| --- | --- | --- |
| Asynq | Go workers, Redis-backed queues, retries, priority queues, timeout/deadline, unique tasks, queue UI. | Do not make Redis queue state the only source of truth for business jobs. |
| River | Go + PostgreSQL jobs, transactional enqueue with domain writes, unique jobs, scheduled/cron jobs, cancellation, job UI. | Watch PostgreSQL queue load if video polling volume grows. |
| Temporal | Durable execution, retries, long-running workflow history, worker model, workflow UI. | Too heavy for the first product slice unless the team already operates Temporal. |
| Hatchet | Postgres-based durable task queue, DAGs, concurrency/rate limits, durable sleeps, event waits, dashboard. | Good future candidate, but adds another platform surface. |
| Inngest | Event triggers, step functions, flow control, concurrency, throttling, state store, dashboard. | More natural for TypeScript/serverless; Go support exists but adds platform coupling. |
| LangGraph | Stateful agent graphs, human-in-the-loop interrupts, durable agent state, observability. | Use as design inspiration, not as the first Go backend runtime dependency. |
| AIYOU | Typed graph nodes, dependency validation, topological execution, provider submit/poll, storyboard child results. | Avoid putting orchestration state only in React/Zustand. |

## Recommendation

Use **PostgreSQL as the workflow and job source of truth**. Queue infrastructure should be replaceable.

Recommended MVP default:

```text
Go API
  -> PostgreSQL domain transaction
  -> River job enqueue in the same transaction
  -> Go worker executes node/job
  -> provider adapter submit/poll
  -> assets + lineage + cost ledger
  -> SSE/WebSocket status events
  -> React Agent Board / Storyboard Kanban
```

Why River-first for MVP:

- Manmu already uses PostgreSQL as source of truth.
- Generation jobs are tightly coupled to project, shot, asset, prompt, approval, and cost rows.
- Transactional enqueue prevents "DB row created but queue message missing" and "queue message exists but DB transaction rolled back" bugs.
- Redis can still be used later for cache, rate counters, realtime fanout, or Asynq if the team prefers Redis-backed queue operations.

Fallback if the team prefers Asynq:

- keep the same domain tables and state machines,
- enqueue Asynq tasks only after committing domain rows,
- reconcile periodically from `generation_jobs` and provider task IDs,
- never let Asynq task state replace `generation_jobs.status`.

## Cross-layer status contract

The frontend should never infer product status from queue internals.

```text
Provider raw status
  -> provider adapter normalization
  -> generation_jobs.status + generation_job_events
  -> workflow_node_runs.status
  -> workflow_runs.status
  -> API DTO / SSE event
  -> React Agent Board / Storyboard Kanban / task queue
```

Contracts:

- API status values are stable typed enums.
- Provider-specific status strings stay in `provider_status` or event metadata.
- Dates are ISO-8601 strings at API boundaries.
- Progress is optional but, when present, uses `0..100`.
- Error responses expose `error_code`, `message`, and `retryable`; secrets and provider credentials are never returned.

## Workflow run state machine

`workflow_runs.status`:

| Status | Meaning | Allowed next statuses |
| --- | --- | --- |
| `draft` | Created but not ready to execute. | `ready`, `canceled` |
| `ready` | Inputs validated; can be started. | `running`, `canceled` |
| `running` | At least one node is active or queueable. | `waiting_approval`, `paused_budget`, `succeeded`, `failed`, `canceling` |
| `waiting_approval` | Human gate blocks downstream work. | `running`, `failed`, `canceling` |
| `paused_budget` | Budget hard/soft cap blocks new cost. | `running`, `failed`, `canceling` |
| `succeeded` | All required terminal nodes succeeded or were intentionally skipped. | terminal |
| `failed` | Workflow cannot continue without user intervention. | terminal for this run; create retry run or retry nodes |
| `canceling` | Stop requested; workers are draining/canceling provider jobs. | `canceled`, `failed` |
| `canceled` | User/system canceled and no active children remain. | terminal |

Rules:

- `workflow_runs` should store `current_node_key`, but UI should derive phase progress from `workflow_node_runs`.
- A run can have multiple active nodes if the graph allows parallel work.
- Terminal workflow status requires all child nodes/jobs to be terminal.
- Canceling a workflow prevents new nodes from being enqueued.

## Workflow node state machine

`workflow_node_runs.status`:

| Status | Meaning | Allowed next statuses |
| --- | --- | --- |
| `pending` | Created but dependencies are incomplete. | `ready`, `skipped`, `canceled` |
| `ready` | Dependencies satisfied and inputs validated. | `queued`, `waiting_approval`, `skipped`, `canceled` |
| `queued` | Worker task enqueued. | `running`, `canceled`, `failed` |
| `running` | Handler is executing. | `waiting_approval`, `waiting_job`, `succeeded`, `failed`, `canceling` |
| `waiting_job` | Node spawned one or more provider jobs and is waiting. | `running`, `succeeded`, `failed`, `canceling` |
| `waiting_approval` | Human gate is pending. | `running`, `failed`, `canceling` |
| `succeeded` | Node output artifacts were persisted. | terminal |
| `failed` | Handler failed after retry policy or non-retryable error. | terminal unless manually retried |
| `skipped` | Node intentionally skipped by graph condition or human decision. | terminal |
| `canceling` | Stop requested for node and children. | `canceled`, `failed` |
| `canceled` | Node canceled before completion. | terminal |
| `blocked` | Missing human input, locked asset, provider capability, or budget. | `ready`, `failed`, `canceling` |

Rules:

- Node handlers are idempotent. Re-running a node must not duplicate locked artifacts or double-charge cost.
- Node output is always persisted as typed artifacts/assets before marking `succeeded`.
- Expensive video nodes require upstream approvals and budget reservation before enqueue.

## Agent run state machine

`agent_runs.status`:

| Status | Meaning |
| --- | --- |
| `queued` | Agent invocation is waiting for a worker. |
| `running` | Agent prompt/tool planning is executing. |
| `tool_calling` | Agent is invoking internal tools, provider adapters, or retrieval. |
| `waiting_job` | Agent spawned a generation job and waits for output. |
| `waiting_approval` | Agent produced artifact candidates that need human review. |
| `succeeded` | Structured output validated and persisted. |
| `failed` | Agent failed with retry policy exhausted or non-retryable error. |
| `canceled` | Parent workflow/node was canceled. |

Rules:

- Agent output must validate against a node-specific JSON schema.
- Prompt render, model, token/cost metadata, and output artifact IDs are required for traceability.
- Agent messages can be stored for debugging, but business state is stored as structured artifacts.

## Generation job state machine

`generation_jobs.status`:

| Status | Meaning | Allowed next statuses |
| --- | --- | --- |
| `draft` | Row created, not yet budgeted/enqueued. | `preflight`, `canceled` |
| `preflight` | Validate provider capability, inputs, prompt length, safety, and budget. | `queued`, `blocked`, `failed`, `canceled` |
| `queued` | Job reserved and queued for submit worker. | `submitting`, `canceled`, `failed` |
| `submitting` | Worker is sending request to provider. | `submitted`, `failed`, `timed_out`, `canceling` |
| `submitted` | Provider accepted request and returned `provider_task_id`. | `polling`, `failed`, `canceling` |
| `polling` | Poll worker checks provider progress. | `downloading`, `needs_review`, `failed`, `timed_out`, `canceling` |
| `downloading` | Provider output is being copied to object storage. | `postprocessing`, `succeeded`, `failed`, `canceling` |
| `postprocessing` | Thumbnail, transcode, subtitle burn-in, or metadata extraction. | `succeeded`, `failed`, `canceling` |
| `needs_review` | Provider result is complete but requires safety/quality review. | `succeeded`, `failed`, `canceling` |
| `succeeded` | Output assets and lineage persisted. | terminal |
| `blocked` | Missing approval, capability, quota, or budget. | `preflight`, `canceled`, `failed` |
| `failed` | Non-retryable failure or retry exhausted. | terminal unless manually retried as new attempt |
| `timed_out` | Provider or worker deadline exceeded. | terminal unless manually retried |
| `canceling` | User/system requested cancel. | `canceled`, `failed`, `succeeded` |
| `canceled` | Canceled locally and provider is canceled or ignored. | terminal |

Rules:

- Store `provider_task_id` as soon as the provider accepts the job.
- Provider `supports_cancel` controls whether remote cancellation is attempted.
- If provider cancellation is unsupported, mark local state `canceling`, ignore late callback outputs unless user chooses to recover them.
- `succeeded` requires output asset rows, lineage edges, prompt render, and final cost ledger entry.

## Job attempts and retry policy

Add a `job_attempts` table or equivalent event rows:

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

Retry policy by task class:

| Task class | Default attempts | Backoff | Retryable examples | Non-retryable examples |
| --- | ---: | --- | --- | --- |
| LLM/story/prompt | 3 | exponential + jitter | 429, 5xx, timeout | invalid prompt schema, content blocked |
| Image generation | 3 | exponential + jitter | provider busy, timeout | unsupported aspect ratio, invalid reference image |
| Video generation | 2 | long exponential + jitter | provider queue timeout, transient 5xx | duration unsupported, safety rejection |
| Audio/TTS | 3 | exponential + jitter | 429, timeout | unsupported voice/model |
| Export/transcode | 2 | exponential + jitter | worker crash, temporary storage issue | missing source asset, invalid timeline |

Retry rules:

- Never retry validation errors.
- Never charge cost twice for the same provider result.
- Use idempotency keys for submit calls when provider supports them.
- Manual retry creates a new attempt and may optionally clone the previous prompt/params.

## Idempotency and uniqueness

Every generation job should have a deterministic `request_key`.

Suggested key inputs:

```text
organization_id
project_id
episode_id
workflow_node_run_id
task_type
provider_id
model_id
prompt_render_id
input_asset_ids sorted
params canonical JSON
```

Rules:

- `generation_jobs.request_key` should be unique for active jobs unless user explicitly requests a new candidate.
- Candidate generation should include `candidate_group_id` and `candidate_no`.
- Regeneration should create a new request key because prompt, seed, params, or candidate group changes.
- Workers should check persisted state before executing side effects.

## Queue design

Recommended queues:

| Queue | Work | Priority |
| --- | --- | --- |
| `control` | workflow scheduling, approval release, cancellation, reconciliation | highest |
| `agent` | LLM agents and structured artifact generation | high |
| `image` | image generation submit/poll/download | medium |
| `video` | video generation submit/poll/download | medium but low concurrency |
| `audio` | TTS, voice, subtitle alignment, lip-sync | medium |
| `export` | FFmpeg render, package export, PDF/storyboard export | medium |
| `maintenance` | stale job reconciliation, provider quota refresh, cleanup | low |

Concurrency limits:

- per organization,
- per project,
- per provider,
- per model,
- per task type,
- per expensive resource class such as video/export.

Rate limiting:

- Provider adapter owns provider-specific QPS/minute/day limits.
- Cost Controller owns budget limits.
- Workflow scheduler should see both as gates before enqueue.

## Human approval gates

Approval gates are first-class blockers, not comments.

Gate types:

- `story_direction`
- `character_lock`
- `scene_lock`
- `prop_lock`
- `storyboard_approval`
- `video_budget_approval`
- `final_timeline`
- `export_approval`

Rules:

- Gate creation moves the related node to `waiting_approval`.
- Approval writes an `audit_event` and releases downstream nodes.
- Rejection can either fail the node or route to a revision node.
- Changes requested should keep all previous artifacts for comparison.

## Budget and cost control

Cost control must happen before and after provider calls.

Recommended tables:

### cost_budgets

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

- `id`
- `budget_id`
- `generation_job_id`
- `workflow_run_id`
- `amount_cents`
- `status`: `reserved | committed | released | expired`
- `created_at`
- `updated_at`

### cost_ledger event reasons

- `estimate`
- `reserve`
- `commit`
- `release`
- `refund`
- `adjustment`
- `provider_credit`

Budget flow:

```text
Estimate job cost
  -> check budget
  -> reserve estimated amount
  -> enqueue job
  -> provider returns actual usage/cost
  -> commit actual amount
  -> release unused reservation or record overage adjustment
```

Hard cap behavior:

- If `reserved + committed + estimate > limit`, block the job and move it to `blocked`.
- If the job is required for workflow completion, move workflow to `paused_budget`.
- User can approve budget increase or switch to a cheaper model.

Soft cap behavior:

- Create warning event and show it in Agent Board.
- Continue only if the project policy allows soft-cap overage.

Cost Controller Agent responsibilities:

- estimate phase costs before video-heavy stages,
- recommend cheaper provider/model/duration,
- stop generation fanout when budget is low,
- summarize actual cost by episode, model, task type, and candidate group.

## Provider adapter contract

Each provider adapter should normalize:

```text
ValidateCapability(ctx, request) -> capability error or nil
EstimateCost(ctx, request) -> cents range
Submit(ctx, request) -> provider_task_id, initial status
Poll(ctx, provider_task_id) -> normalized status, progress, provider metadata
Cancel(ctx, provider_task_id) -> cancellation result
FetchResult(ctx, provider_task_id) -> downloadable outputs
NormalizeError(err) -> error_code, retryable, user_message
```

Adapter rules:

- Do not leak provider credentials into logs, errors, prompt renders, or frontend DTOs.
- Keep provider raw payloads in debug metadata only when safe.
- Normalize provider capability failures before queueing expensive work.
- Record provider latency and cost metadata on every attempt.

## Reconciliation and late callbacks

Long-running video jobs need reconciliation.

Maintenance jobs:

- find `submitted` or `polling` jobs with stale `updated_at`,
- poll provider again,
- fetch missing outputs,
- mark timed out jobs,
- detect provider completion after local cancel,
- repair node/workflow aggregate status.

Rules:

- Provider callback and poller must be idempotent.
- Late success after local cancel should be stored as an orphaned asset only if policy allows recovery; otherwise keep event metadata and do not attach it to the approved timeline.
- Reconciliation never overwrites locked user choices without an audit event.

## Realtime events

Emit stable events for Studio:

| Event | Payload |
| --- | --- |
| `workflow.status_changed` | workflow id, old status, new status, phase |
| `node.status_changed` | node id, node key, old status, new status |
| `agent.output_created` | agent run id, artifact ids, summary |
| `approval.requested` | gate id, gate type, subject |
| `generation.progress` | job id, task type, status, progress, message |
| `generation.completed` | job id, output asset ids, cost |
| `generation.failed` | job id, error code, retryable, message |
| `cost.warning` | budget id, threshold, committed/reserved/limit |
| `review.issue_created` | issue id, severity, subject, suggested fix |

## MVP workflow execution sequence

```text
1. Create workflow_run from default SOP template.
2. Materialize workflow_node_runs from the template graph.
3. Validate story source and project policy.
4. Run Producer -> Story Analyst -> Screenwriter.
5. Create story approval gate.
6. After approval, run Character / Scene / Prop designers in parallel.
7. Lock selected C/S/P versions through approval gates.
8. Run Storyboard -> Prompt Engineer -> Safety.
9. Estimate video-heavy phase cost and request approval if needed.
10. Enqueue keyframe/video/audio generation jobs.
11. Poll/download/postprocess assets and write lineage.
12. Run Continuity Supervisor and create review issues.
13. Run Editor Agent to assemble rough timeline.
14. Human approves final timeline.
15. Enqueue export job and attach output asset.
```

## Implementation boundary

Suggested Go packages:

```text
internal/workflow
  template.go
  scheduler.go
  state_machine.go
  node_handlers.go
  approvals.go

internal/jobs
  queue.go
  workers.go
  attempts.go
  reconciliation.go

internal/provider
  adapter.go
  llm.go
  image.go
  video.go
  audio.go

internal/cost
  estimator.go
  budget.go
  reservation.go
  ledger.go

internal/realtime
  events.go
  sse.go
```

Rule of thumb:

- `workflow` decides what should happen next.
- `jobs` moves work through queues and attempts.
- `provider` talks to external APIs.
- `cost` blocks or reserves spending.
- `repo` persists all state transitions.
- `realtime` broadcasts stable DTOs.

## MVP database additions

Add to the previous schema draft:

- `workflow_events`
- `generation_job_events`
- `job_attempts`
- `cost_budgets`
- `cost_reservations`
- `provider_rate_limits`
- `workflow_run_locks` only if optimistic locking is not enough.

Indexes:

- `generation_jobs(status, updated_at)`
- `generation_jobs(request_key)`
- `generation_jobs(provider_id, provider_task_id)`
- `workflow_node_runs(workflow_run_id, status)`
- `cost_reservations(budget_id, status)`
- `job_attempts(generation_job_id, attempt_no)`

## Open decisions

- Confirm River-first vs Asynq-first queue implementation.
- Decide whether `cost_cents` should be stored as integer cents only or decimal minor units for non-USD currencies.
- Decide whether the first release needs organization-level monthly budgets or only project/episode budgets.
- Decide whether to build SSE first or WebSocket first for Studio realtime.
