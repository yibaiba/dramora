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

### Scenario: Production core persistence

#### 1. Scope / Trigger

- Trigger: project/episode CRUD and production read APIs introduced PostgreSQL migrations and pgx repositories.

#### 2. Signatures

Migration files:

```text
db/migrations/000001_create_identity.up.sql
db/migrations/000002_create_projects.up.sql
db/migrations/000003_create_production_core.up.sql
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
- Nullable UUIDs exposed to API read models are normalized to empty string until typed nullable DTOs are introduced.

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
