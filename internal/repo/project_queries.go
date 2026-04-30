package repo

const listProjectsSQL = `
SELECT id::text, organization_id::text, name, description, status, created_at, updated_at
FROM projects
WHERE organization_id = $1::uuid
ORDER BY created_at DESC, id
`

const createProjectSQL = `
INSERT INTO projects (id, organization_id, name, description, status)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, organization_id::text, name, description, status, created_at, updated_at
`

const getProjectSQL = `
SELECT id::text, organization_id::text, name, description, status, created_at, updated_at
FROM projects
WHERE id = $1::uuid AND organization_id = $2::uuid
`

const lookupProjectByIDSQL = `
SELECT id::text, organization_id::text, name, description, status, created_at, updated_at
FROM projects
WHERE id = $1::uuid
`

const listEpisodesSQL = `
SELECT id::text, project_id::text, number, title, status, created_at, updated_at
FROM episodes
WHERE project_id = $1::uuid
ORDER BY number, created_at, id
`

const createEpisodeSQL = `
INSERT INTO episodes (id, project_id, number, title, status)
VALUES ($1::uuid, $2::uuid, $3, $4, $5)
RETURNING id::text, project_id::text, number, title, status, created_at, updated_at
`

const getEpisodeSQL = `
SELECT id::text, project_id::text, number, title, status, created_at, updated_at
FROM episodes
WHERE id = $1::uuid
`
