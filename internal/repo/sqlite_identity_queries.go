package repo

const sqliteCreateUserSQL = `
INSERT INTO users (id, email, display_name, password_hash)
VALUES (?, ?, ?, ?)
`

const sqliteCreateOrganizationMemberSQL = `
INSERT INTO organization_members (organization_id, user_id, role)
VALUES (?, ?, ?)
`

const sqliteAuthIdentitySelect = `
SELECT
    users.id,
    users.email,
    users.display_name,
    users.password_hash,
    organization_members.organization_id,
    organization_members.role,
    users.created_at,
    users.updated_at
FROM users
JOIN organization_members ON organization_members.user_id = users.id
`

const sqliteGetAuthIdentityByEmailSQL = sqliteAuthIdentitySelect + `
WHERE lower(users.email) = lower(?)
ORDER BY organization_members.created_at ASC
LIMIT 1
`

const sqliteGetAuthIdentityByUserIDSQL = sqliteAuthIdentitySelect + `
WHERE users.id = ?
ORDER BY organization_members.created_at ASC
LIMIT 1
`
