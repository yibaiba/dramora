package repo

const createUserSQL = `
INSERT INTO users (id, email, display_name, password_hash)
VALUES ($1::uuid, $2, $3, $4)
`

const createOrganizationMemberSQL = `
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1::uuid, $2::uuid, $3)
`

const authIdentitySelect = `
SELECT
    users.id::text,
    users.email,
    users.display_name,
    users.password_hash,
    organization_members.organization_id::text,
    organization_members.role,
    users.created_at,
    users.updated_at
FROM users
JOIN organization_members ON organization_members.user_id = users.id
`

const getAuthIdentityByEmailSQL = authIdentitySelect + `
WHERE lower(users.email) = lower($1)
ORDER BY organization_members.created_at ASC
LIMIT 1
`

const getAuthIdentityByUserIDSQL = authIdentitySelect + `
WHERE users.id = $1::uuid
ORDER BY organization_members.created_at ASC
LIMIT 1
`
