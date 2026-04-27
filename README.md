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
The worker uses the same repository wiring. With PostgreSQL configured, it can no-op process queued `generation_jobs` through the current status machine so workflow plumbing is testable before real model providers are attached.

Example local environment:

```bash
export MANMU_DATABASE_URL='postgres://manmu:manmu@localhost:5432/manmu?sslmode=disable'
export MANMU_DEFAULT_ORGANIZATION_ID='00000000-0000-0000-0000-000000000001'
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
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-map:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/story-map
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/storyboard-shots:seed
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/storyboard-shots
curl -sS http://127.0.0.1:8080/api/v1/workflow-runs/{workflowRunId}
curl -sS http://127.0.0.1:8080/api/v1/episodes/{episodeId}/timeline
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/timeline \
  -H 'content-type: application/json' \
  -d '{"duration_ms":15000}'
curl -sS -X POST http://127.0.0.1:8080/api/v1/episodes/{episodeId}/exports
curl -sS http://127.0.0.1:8080/api/v1/exports/{exportId}
```

Core production migrations now include story analysis artifacts, character/scene/prop maps, storyboard shot cards, assets, artifact lineage, workflow runs, workflow node runs, generation jobs/events, timelines/tracks/clips, and exports.

## Studio frontend

The first React Studio slice lives in `apps/studio`:

- Vite + React + TypeScript.
- TanStack Query for server state.
- Zustand for selected project and local Studio event state.
- Lucide icons for structural icons.
- Tailwind v4 Vite plugin plus custom design tokens.
- Vite dev proxy maps `/api`, `/healthz`, and `/readyz` to `http://127.0.0.1:8080`.

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

Validate the Studio:

```bash
cd apps/studio
npm run lint
npm run build
```
