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

const sqliteCreateOrganizationSQL = `
INSERT INTO organizations (id, name) VALUES (?, ?)
`

const sqliteInvitationSelect = `
SELECT
    id,
    organization_id,
    email,
    role,
    token,
    invited_by_user_id,
    status,
    expires_at,
    accepted_at,
    accepted_by_user_id,
    created_at,
    updated_at
FROM organization_invitations
`

const sqliteCreateInvitationSQL = `
INSERT INTO organization_invitations
    (id, organization_id, email, role, token, invited_by_user_id, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

const sqliteGetInvitationByTokenSQL = sqliteInvitationSelect + `WHERE token = ?`

const sqliteGetInvitationByIDSQL = sqliteInvitationSelect + `WHERE id = ?`

const sqliteMarkInvitationAcceptedSQL = `
UPDATE organization_invitations
SET status = 'accepted',
    accepted_at = ?,
    accepted_by_user_id = ?,
    updated_at = ?
WHERE id = ? AND status = 'pending'
`

const sqliteListInvitationsByOrgSQL = sqliteInvitationSelect + `
WHERE organization_id = ?
ORDER BY created_at DESC
`

const sqliteRevokeInvitationSQL = `
UPDATE organization_invitations
SET status = 'revoked',
    updated_at = ?
WHERE id = ? AND organization_id = ? AND status = 'pending'
`
