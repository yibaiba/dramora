-- name: ListProjects :many
SELECT id::text, organization_id::text, name, description, status, created_at, updated_at
FROM projects
WHERE organization_id = $1::uuid
ORDER BY created_at DESC, id;

-- name: CreateProject :one
INSERT INTO projects (id, organization_id, name, description, status)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, organization_id::text, name, description, status, created_at, updated_at;

-- name: GetProject :one
SELECT id::text, organization_id::text, name, description, status, created_at, updated_at
FROM projects
WHERE id = $1::uuid AND organization_id = $2::uuid;

-- name: ListEpisodes :many
SELECT id::text, project_id::text, number, title, status, created_at, updated_at
FROM episodes
WHERE project_id = $1::uuid
ORDER BY number, created_at, id;

-- name: CreateEpisode :one
INSERT INTO episodes (id, project_id, number, title, status)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, project_id::text, number, title, status, created_at, updated_at;

-- name: GetEpisode :one
SELECT id::text, project_id::text, number, title, status, created_at, updated_at
FROM episodes
WHERE id = $1::uuid;
