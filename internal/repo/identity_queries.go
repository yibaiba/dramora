package repo

const createUserSQL = `
INSERT INTO users (id, email, display_name, password_hash)
VALUES ($1::uuid, $2, $3, $4)
`

const createOrganizationMemberSQL = `
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1::uuid, $2::uuid, $3)
`

const organizationMemberSelect = `
SELECT
    organization_members.organization_id::text,
    users.id::text,
    users.email,
    users.display_name,
    organization_members.role,
    organization_members.created_at,
    users.updated_at
FROM organization_members
JOIN users ON users.id = organization_members.user_id
`

const listOrganizationMembersSQL = organizationMemberSelect + `
WHERE organization_members.organization_id = $1::uuid
ORDER BY
    CASE organization_members.role
        WHEN 'owner' THEN 0
        WHEN 'admin' THEN 1
        WHEN 'editor' THEN 2
        ELSE 3
    END,
    lower(users.email) ASC
`

const getOrganizationMemberSQL = organizationMemberSelect + `
WHERE organization_members.organization_id = $1::uuid
  AND users.id = $2::uuid
LIMIT 1
`

const updateOrganizationMemberRoleSQL = `
UPDATE organization_members
SET role = $3
WHERE organization_id = $1::uuid
  AND user_id = $2::uuid
`

const removeOrganizationMemberSQL = `
DELETE FROM organization_members
WHERE organization_id = $1::uuid
  AND user_id = $2::uuid
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

const updateUserPasswordHashSQL = `
UPDATE users
SET password_hash = $2,
    updated_at = $3
WHERE id = $1::uuid
`

const userAPIKeySelect = `
SELECT
    id::text,
    user_id::text,
    name,
    token_preview,
    scope,
    is_active,
    expires_at,
    last_used_at,
    created_at,
    updated_at
FROM user_api_keys
`

const listUserAPIKeysSQL = userAPIKeySelect + `
WHERE user_id = $1::uuid
ORDER BY created_at DESC
`

const createUserAPIKeySQL = `
INSERT INTO user_api_keys (
    id, user_id, name, token_hash, token_preview,
    scope, is_active, expires_at, created_at, updated_at
) VALUES (
    $1::uuid, $2::uuid, $3, $4, $5,
    $6, $7, $8, $9, $10
)
RETURNING
    id::text,
    user_id::text,
    name,
    token_preview,
    scope,
    is_active,
    expires_at,
    last_used_at,
    created_at,
    updated_at
`

const updateUserAPIKeySQL = `
UPDATE user_api_keys
SET name = $3,
    scope = $4,
    expires_at = $5,
    updated_at = $6
WHERE id = $1::uuid
  AND user_id = $2::uuid
RETURNING
    id::text,
    user_id::text,
    name,
    token_preview,
    scope,
    is_active,
    expires_at,
    last_used_at,
    created_at,
    updated_at
`

const setUserAPIKeyActiveSQL = `
UPDATE user_api_keys
SET is_active = $3,
    updated_at = $4
WHERE id = $1::uuid
  AND user_id = $2::uuid
RETURNING
    id::text,
    user_id::text,
    name,
    token_preview,
    scope,
    is_active,
    expires_at,
    last_used_at,
    created_at,
    updated_at
`

const deleteUserAPIKeySQL = `
DELETE FROM user_api_keys
WHERE id = $1::uuid
  AND user_id = $2::uuid
`

const createOrganizationSQL = `
INSERT INTO organizations (id, name)
VALUES ($1::uuid, $2)
`

const invitationSelect = `
SELECT
    id::text,
    organization_id::text,
    email,
    role,
    token,
    invited_by_user_id::text,
    status,
    expires_at,
    accepted_at,
    accepted_by_user_id::text,
    created_at,
    updated_at
FROM organization_invitations
`

const createInvitationSQL = `
INSERT INTO organization_invitations
    (id, organization_id, email, role, token, invited_by_user_id, expires_at)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7)
RETURNING
    id::text,
    organization_id::text,
    email,
    role,
    token,
    invited_by_user_id::text,
    status,
    expires_at,
    accepted_at,
    accepted_by_user_id::text,
    created_at,
    updated_at
`

const getInvitationByTokenSQL = invitationSelect + `WHERE token = $1`

const markInvitationAcceptedSQL = `
UPDATE organization_invitations
SET status = 'accepted',
    accepted_at = $3,
    accepted_by_user_id = $2::uuid,
    updated_at = $3
WHERE id = $1::uuid AND status = 'pending'
`

const listInvitationsByOrgSQL = invitationSelect + `
WHERE organization_id = $1::uuid
ORDER BY created_at DESC
`

const revokeInvitationSQL = `
UPDATE organization_invitations
SET status = 'revoked',
    updated_at = $3
WHERE id = $1::uuid AND organization_id = $2::uuid AND status = 'pending'
`

const insertInvitationAuditEventSQL = `
INSERT INTO organization_invitation_events (
    id, organization_id, invitation_id, action,
    actor_user_id, actor_email, email, role, note, created_at
) VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4,
    NULLIF($5, '')::uuid, NULLIF($6, ''), $7, $8, NULLIF($9, ''), $10
)
`
