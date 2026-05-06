package repo

const sqliteCreateUserSQL = `
INSERT INTO users (id, email, display_name, password_hash)
VALUES (?, ?, ?, ?)
`

const sqliteCreateOrganizationMemberSQL = `
INSERT INTO organization_members (organization_id, user_id, role)
VALUES (?, ?, ?)
`

const sqliteOrganizationMemberSelect = `
SELECT
    organization_members.organization_id,
    users.id,
    users.email,
    users.display_name,
    organization_members.role,
    organization_members.created_at,
    users.updated_at
FROM organization_members
JOIN users ON users.id = organization_members.user_id
`

const sqliteListOrganizationMembersSQL = sqliteOrganizationMemberSelect + `
WHERE organization_members.organization_id = ?
ORDER BY
    CASE organization_members.role
        WHEN 'owner' THEN 0
        WHEN 'admin' THEN 1
        WHEN 'editor' THEN 2
        ELSE 3
    END,
    lower(users.email) ASC
`

const sqliteGetOrganizationMemberSQL = sqliteOrganizationMemberSelect + `
WHERE organization_members.organization_id = ?
  AND users.id = ?
LIMIT 1
`

const sqliteUpdateOrganizationMemberRoleSQL = `
UPDATE organization_members
SET role = ?
WHERE organization_id = ?
  AND user_id = ?
`

const sqliteRemoveOrganizationMemberSQL = `
DELETE FROM organization_members
WHERE organization_id = ?
  AND user_id = ?
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

const sqliteUpdateUserPasswordHashSQL = `
UPDATE users
SET password_hash = ?,
    updated_at = ?
WHERE id = ?
`

const sqliteUserAPIKeySelect = `
SELECT
    id,
    user_id,
    name,
    token_preview,
    scope,
    is_active,
    COALESCE(expires_at, ''),
    COALESCE(last_used_at, ''),
    created_at,
    updated_at
FROM user_api_keys
`

const sqliteListUserAPIKeysSQL = sqliteUserAPIKeySelect + `
WHERE user_id = ?
ORDER BY created_at DESC
`

const sqliteGetUserAPIKeyByIDSQL = sqliteUserAPIKeySelect + `
WHERE id = ?
  AND user_id = ?
LIMIT 1
`

const sqliteCreateUserAPIKeySQL = `
INSERT INTO user_api_keys (
    id, user_id, name, token_hash, token_preview,
    scope, is_active, expires_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const sqliteUpdateUserAPIKeySQL = `
UPDATE user_api_keys
SET name = ?,
    scope = ?,
    expires_at = ?,
    updated_at = ?
WHERE id = ?
  AND user_id = ?
`

const sqliteSetUserAPIKeyActiveSQL = `
UPDATE user_api_keys
SET is_active = ?,
    updated_at = ?
WHERE id = ?
  AND user_id = ?
`

const sqliteDeleteUserAPIKeySQL = `
DELETE FROM user_api_keys
WHERE id = ?
  AND user_id = ?
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

const sqliteInsertInvitationAuditEventSQL = `
INSERT INTO organization_invitation_events (
    id, organization_id, invitation_id, action,
    actor_user_id, actor_email, email, role, note, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
