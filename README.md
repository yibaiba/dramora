# dramora

Manmu AI Manju Studio scaffold.

## Backend

The first backend slice is a Go modular monolith:

- `apps/api`: HTTP API entrypoint.
- `apps/worker`: background worker entrypoint.
- `internal/httpapi`: Chi router, middleware, responses, SSE placeholder.
- `internal/domain`: core status enums and transition validators.
- `internal/repo`: repository and transaction boundaries.
- `internal/workflow`: typed workflow graph primitives.
- `internal/jobs`: worker/job type scaffolding.
- `api/openapi.yaml`: initial API contract stub.
- `db/migrations`: PostgreSQL migrations.
- `db/queries`: sqlc query source files.

Run the API:

```bash
go run ./apps/api
```

In local mode, the API also runs an inline worker by default so Studio can auto-complete queued story analysis and export jobs. Set `MANMU_INLINE_WORKER=false` to disable that behavior, or set it to `true` outside local mode when you intentionally want one process to run both API and worker loops.

Run the worker:

```bash
go run ./apps/worker
```

Run tests:

```bash
go test ./...
```

The scaffold does not require provider secrets or a database connection yet.

If `MANMU_DATABASE_URL` is empty, the API uses an in-memory repository for local smoke tests. Set `MANMU_DATABASE_URL` after applying migrations to use PostgreSQL-backed CRUD.
The worker uses the same repository wiring. With PostgreSQL configured, it can no-op process queued `generation_jobs` and `exports` through their current status machines so workflow plumbing is testable before real model providers or renderers are attached.
SD2/Seedance fast prompt packs work without provider secrets. The Seedance adapter defaults to fake mode and switches to Ark request mode only when `ARK_API_KEY` is present at runtime.

Example local environment:

```bash
export MANMU_DATABASE_URL='postgres://manmu:manmu@localhost:5432/manmu?sslmode=disable'
export MANMU_DEFAULT_ORGANIZATION_ID='00000000-0000-0000-0000-000000000001'
# Optional: enables real Ark submission mode in the Seedance adapter boundary.
export ARK_API_KEY='...'
```

Apply migrations with your migration runner of choice using files in `db/migrations`.

Project and episode CRUD routes:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/projects \
  -H 'content-type: application/json' \
  -d '{"name":"漫幕","description":"AI manju studio"}'

curl -sS http://127.0.0.1:8080/api/v1/projects
```

Production foundation routes:

```bash
curl -sS http://127.0.0.1:8080/api/v1/generation-jobs
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-analysis/start
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-analyses
curl -sS http://127.0.0.1:8080/api/v1/story-analyses/{analysisId}
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/approval-gates:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/approval-gates
curl -sS -X POST http://127.0.0.1:8080/api/v1/approval-gates/{gateId}:approve \
  -H 'content-type: application/json' \
  -d '{"reviewed_by":"studio","review_note":"approved"}'
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-map:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-map
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/storyboard-shots:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/storyboard-shots
curl -sS -X POST http://127.0.0.1:8080/api/v1/storyboard-shots/{shotId}/prompt-pack:generate
curl -sS http://127.0.0.1:8080/api/v1/storyboard-shots/{shotId}/prompt-pack
curl -sS -X POST http://127.0.0.1:8080/api/v1/storyboard-shots/{shotId}/videos:generate
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/assets:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/assets
curl -sS -X POST http://127.0.0.1:8080/api/v1/assets/{assetId}:lock
curl -sS http://127.0.0.1:8080/api/v1/workflow-runs/{workflowRunId}
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/timeline
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/timeline \
  -H 'content-type: application/json' \
  -d '{"duration_ms":15000}'
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/exports
curl -sS http://127.0.0.1:8080/api/v1/exports/{exportId}
```

Core production migrations now include story analysis artifacts, human approval gates, character/scene/prop maps, storyboard shot cards, SD2 prompt packs, assets, artifact lineage, workflow runs, workflow node runs, generation jobs/events, timelines/tracks/clips, and exports.

## Studio frontend

The first React Studio slice lives in `apps/studio`:

- Vite + React + TypeScript.
- TanStack Query for server state.
- Zustand for selected project and local Studio event state.
- Lucide icons for structural icons.
- Tailwind v4 Vite plugin plus custom design tokens.
- Vite dev proxy maps `/api`, `/healthz`, and `/readyz` to `http://127.0.0.1:8080` by default.

Run the Studio:

```bash
cd apps/studio
npm install
npm run dev
```

Point Studio directly at another API origin if needed:

```bash
VITE_MANMU_API_BASE_URL=http://127.0.0.1:8080 npm run dev
```

Or keep relative API paths and point the Vite dev proxy at another backend port:

```bash
VITE_MANMU_API_PROXY_TARGET=http://127.0.0.1:16666 npm run dev -- --port 16667
```

Validate the Studio:

```bash
cd apps/studio
npm run lint
npm run build
```
