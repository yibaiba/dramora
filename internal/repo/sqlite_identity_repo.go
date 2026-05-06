package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type SQLiteIdentityRepository struct {
	db *sql.DB
}

func NewSQLiteIdentityRepository(db *sql.DB) *SQLiteIdentityRepository {
	return &SQLiteIdentityRepository{db: db}
}

func (r *SQLiteIdentityRepository) CreateUserWithMembership(
	ctx context.Context,
	params CreateUserWithMembershipParams,
) (AuthIdentity, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, sqliteCreateUserSQL,
		params.UserID,
		params.Email,
		params.DisplayName,
		params.PasswordHash,
	)
	if err != nil {
		if isSQLiteUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		return AuthIdentity{}, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqliteCreateOrganizationMemberSQL, params.OrganizationID, params.UserID, params.Role)
	if err != nil {
		if isSQLiteFKViolation(err) {
			return AuthIdentity{}, domain.ErrNotFound
		}
		if isSQLiteUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		return AuthIdentity{}, fmt.Errorf("create organization member: %w", err)
	}

	identity, err := scanSQLiteAuthIdentity(tx.QueryRowContext(ctx, sqliteGetAuthIdentityByUserIDSQL, params.UserID))
	if err == sql.ErrNoRows {
		return AuthIdentity{}, domain.ErrNotFound
	}
	if err != nil {
		return AuthIdentity{}, err
	}
	if err := tx.Commit(); err != nil {
		return AuthIdentity{}, fmt.Errorf("commit create user: %w", err)
	}
	return identity, nil
}

func (r *SQLiteIdentityRepository) GetAuthIdentityByEmail(ctx context.Context, email string) (AuthIdentity, error) {
	identity, err := scanSQLiteAuthIdentity(r.db.QueryRowContext(ctx, sqliteGetAuthIdentityByEmailSQL, email))
	if err == sql.ErrNoRows {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func (r *SQLiteIdentityRepository) GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error) {
	identity, err := scanSQLiteAuthIdentity(r.db.QueryRowContext(ctx, sqliteGetAuthIdentityByUserIDSQL, userID))
	if err == sql.ErrNoRows {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func scanSQLiteAuthIdentity(scanner sqliteScanner) (AuthIdentity, error) {
	var (
		identity  AuthIdentity
		createdAt string
		updatedAt string
	)
	err := scanner.Scan(
		&identity.User.ID,
		&identity.User.Email,
		&identity.User.DisplayName,
		&identity.PasswordHash,
		&identity.OrganizationID,
		&identity.Role,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return AuthIdentity{}, err
	}
	if identity.User.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return AuthIdentity{}, err
	}
	if identity.User.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return AuthIdentity{}, err
	}
	return identity, nil
}

func (r *SQLiteIdentityRepository) UpdateUserPasswordHash(
	ctx context.Context,
	userID, passwordHash string,
	updatedAt time.Time,
) error {
	res, err := r.db.ExecContext(
		ctx,
		sqliteUpdateUserPasswordHashSQL,
		passwordHash,
		updatedAt.UTC().Format(time.RFC3339Nano),
		userID,
	)
	if err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update user password hash rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteIdentityRepository) ListUserAPIKeys(ctx context.Context, userID string) ([]domain.UserAPIKey, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListUserAPIKeysSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}
	defer rows.Close()
	out := make([]domain.UserAPIKey, 0)
	for rows.Next() {
		item, scanErr := scanSQLiteUserAPIKey(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan user api key: %w", scanErr)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *SQLiteIdentityRepository) CreateUserAPIKey(
	ctx context.Context,
	params CreateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	_, err := r.db.ExecContext(
		ctx,
		sqliteCreateUserAPIKeySQL,
		params.KeyID,
		params.UserID,
		params.Name,
		params.TokenHash,
		params.TokenPreview,
		params.Scope,
		boolToSQLiteInt(params.IsActive),
		nullableTimeString(params.ExpiresAt),
		params.CreatedAt.UTC().Format(time.RFC3339Nano),
		params.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		if isSQLiteFKViolation(err) {
			return domain.UserAPIKey{}, domain.ErrNotFound
		}
		if isSQLiteUniqueViolation(err) {
			return domain.UserAPIKey{}, domain.ErrInvalidInput
		}
		return domain.UserAPIKey{}, fmt.Errorf("create user api key: %w", err)
	}
	return r.getUserAPIKeyByID(ctx, params.UserID, params.KeyID)
}

func (r *SQLiteIdentityRepository) UpdateUserAPIKey(
	ctx context.Context,
	params UpdateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	res, err := r.db.ExecContext(
		ctx,
		sqliteUpdateUserAPIKeySQL,
		params.Name,
		params.Scope,
		nullableTimeString(params.ExpiresAt),
		params.UpdatedAt.UTC().Format(time.RFC3339Nano),
		params.KeyID,
		params.UserID,
	)
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("update user api key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("update user api key rows affected: %w", err)
	}
	if affected == 0 {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	return r.getUserAPIKeyByID(ctx, params.UserID, params.KeyID)
}

func (r *SQLiteIdentityRepository) SetUserAPIKeyActive(
	ctx context.Context,
	userID, keyID string,
	isActive bool,
	updatedAt time.Time,
) (domain.UserAPIKey, error) {
	res, err := r.db.ExecContext(
		ctx,
		sqliteSetUserAPIKeyActiveSQL,
		boolToSQLiteInt(isActive),
		updatedAt.UTC().Format(time.RFC3339Nano),
		keyID,
		userID,
	)
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("set user api key active: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("set user api key active rows affected: %w", err)
	}
	if affected == 0 {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	return r.getUserAPIKeyByID(ctx, userID, keyID)
}

func (r *SQLiteIdentityRepository) DeleteUserAPIKey(ctx context.Context, userID, keyID string) error {
	res, err := r.db.ExecContext(ctx, sqliteDeleteUserAPIKeySQL, keyID, userID)
	if err != nil {
		return fmt.Errorf("delete user api key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user api key rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteIdentityRepository) ListOrganizationMembers(
	ctx context.Context,
	organizationID string,
) ([]domain.OrganizationMember, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListOrganizationMembersSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list organization members: %w", err)
	}
	defer rows.Close()
	out := make([]domain.OrganizationMember, 0)
	for rows.Next() {
		member, scanErr := scanOrganizationMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan organization member: %w", scanErr)
		}
		out = append(out, member)
	}
	return out, rows.Err()
}

func (r *SQLiteIdentityRepository) GetOrganizationMember(
	ctx context.Context,
	organizationID, userID string,
) (domain.OrganizationMember, error) {
	member, err := scanOrganizationMember(r.db.QueryRowContext(ctx, sqliteGetOrganizationMemberSQL, organizationID, userID))
	if err == sql.ErrNoRows {
		return domain.OrganizationMember{}, domain.ErrNotFound
	}
	return member, err
}

func (r *SQLiteIdentityRepository) UpdateOrganizationMemberRole(
	ctx context.Context,
	organizationID, userID, role string,
) error {
	res, err := r.db.ExecContext(ctx, sqliteUpdateOrganizationMemberRoleSQL, role, organizationID, userID)
	if err != nil {
		return fmt.Errorf("update organization member role: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update organization member role rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteIdentityRepository) RemoveOrganizationMember(
	ctx context.Context,
	organizationID, userID string,
) error {
	res, err := r.db.ExecContext(ctx, sqliteRemoveOrganizationMemberSQL, organizationID, userID)
	if err != nil {
		return fmt.Errorf("remove organization member: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("remove organization member rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteIdentityRepository) CreateOrganization(ctx context.Context, params CreateOrganizationParams) error {
	if _, err := r.db.ExecContext(ctx, sqliteCreateOrganizationSQL, params.OrganizationID, params.Name); err != nil {
		if isSQLiteUniqueViolation(err) {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *SQLiteIdentityRepository) CreateInvitation(
	ctx context.Context,
	params CreateInvitationParams,
) (domain.OrganizationInvitation, error) {
	var invitedBy any
	if params.InvitedByUserID != "" {
		invitedBy = params.InvitedByUserID
	}
	if _, err := r.db.ExecContext(ctx, sqliteCreateInvitationSQL,
		params.InvitationID,
		params.OrganizationID,
		params.Email,
		params.Role,
		params.Token,
		invitedBy,
		params.ExpiresAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		if isSQLiteUniqueViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrInvalidInput
		}
		if isSQLiteFKViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrNotFound
		}
		return domain.OrganizationInvitation{}, fmt.Errorf("create invitation: %w", err)
	}
	return scanSQLiteInvitation(r.db.QueryRowContext(ctx, sqliteGetInvitationByIDSQL, params.InvitationID))
}

func (r *SQLiteIdentityRepository) GetInvitationByToken(ctx context.Context, token string) (domain.OrganizationInvitation, error) {
	inv, err := scanSQLiteInvitation(r.db.QueryRowContext(ctx, sqliteGetInvitationByTokenSQL, token))
	if err == sql.ErrNoRows {
		return domain.OrganizationInvitation{}, domain.ErrNotFound
	}
	return inv, err
}

func (r *SQLiteIdentityRepository) MarkInvitationAccepted(ctx context.Context, invitationID, userID string, acceptedAt time.Time) error {
	stamp := acceptedAt.UTC().Format(time.RFC3339Nano)
	res, err := r.db.ExecContext(ctx, sqliteMarkInvitationAcceptedSQL, stamp, userID, stamp, invitationID)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteIdentityRepository) ListOrganizationInvitations(ctx context.Context, organizationID string) ([]domain.OrganizationInvitation, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListInvitationsByOrgSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()
	var out []domain.OrganizationInvitation
	for rows.Next() {
		inv, scanErr := scanSQLiteInvitation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan invitation: %w", scanErr)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *SQLiteIdentityRepository) RevokeInvitation(ctx context.Context, invitationID, organizationID string, revokedAt time.Time) error {
	res, err := r.db.ExecContext(ctx, sqliteRevokeInvitationSQL, revokedAt.UTC(), invitationID, organizationID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke invitation rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTimeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func sqliteIntToBool(value int) bool {
	return value != 0
}

func scanSQLiteUserAPIKey(scanner sqliteScanner) (domain.UserAPIKey, error) {
	var item domain.UserAPIKey
	var expiresAt string
	var lastUsedAt string
	var isActive int
	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.TokenPreview,
		&item.Scope,
		&isActive,
		&expiresAt,
		&lastUsedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.UserAPIKey{}, err
	}
	item.IsActive = sqliteIntToBool(isActive)
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	if expiresAt != "" {
		t, parseErr := parseSQLiteTime(expiresAt)
		if parseErr != nil {
			return domain.UserAPIKey{}, fmt.Errorf("parse api key expires_at: %w", parseErr)
		}
		item.ExpiresAt = &t
	}
	if lastUsedAt != "" {
		t, parseErr := parseSQLiteTime(lastUsedAt)
		if parseErr != nil {
			return domain.UserAPIKey{}, fmt.Errorf("parse api key last_used_at: %w", parseErr)
		}
		item.LastUsedAt = &t
	}
	return item, nil
}

func (r *SQLiteIdentityRepository) getUserAPIKeyByID(
	ctx context.Context,
	userID, keyID string,
) (domain.UserAPIKey, error) {
	item, err := scanSQLiteUserAPIKey(r.db.QueryRowContext(ctx, sqliteGetUserAPIKeyByIDSQL, keyID, userID))
	if err == sql.ErrNoRows {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("get user api key by id: %w", err)
	}
	return item, nil
}

func (r *SQLiteIdentityRepository) AppendInvitationAuditEvent(
	ctx context.Context,
	params AppendInvitationAuditParams,
) (domain.InvitationAuditEvent, error) {
	createdAt := params.CreatedAt.UTC()
	if _, err := r.db.ExecContext(ctx, sqliteInsertInvitationAuditEventSQL,
		params.EventID,
		params.OrganizationID,
		params.InvitationID,
		params.Action,
		nullableString(params.ActorUserID),
		nullableString(params.ActorEmail),
		params.Email,
		params.Role,
		nullableString(params.Note),
		createdAt.Format("2006-01-02T15:04:05.000Z"),
	); err != nil {
		return domain.InvitationAuditEvent{}, fmt.Errorf("append invitation audit: %w", err)
	}
	return domain.InvitationAuditEvent{
		ID:             params.EventID,
		OrganizationID: params.OrganizationID,
		InvitationID:   params.InvitationID,
		Action:         params.Action,
		ActorUserID:    params.ActorUserID,
		ActorEmail:     params.ActorEmail,
		Email:          params.Email,
		Role:           params.Role,
		Note:           params.Note,
		CreatedAt:      createdAt,
	}, nil
}

func (r *SQLiteIdentityRepository) ListInvitationAuditEvents(
	ctx context.Context,
	filter InvitationAuditFilter,
) (InvitationAuditPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args := []any{filter.OrganizationID}
	clauses := []string{"organization_id = ?"}
	if len(filter.Actions) > 0 {
		placeholders := make([]string, len(filter.Actions))
		for i, action := range filter.Actions {
			placeholders[i] = "?"
			args = append(args, action)
		}
		clauses = append(clauses, "action IN ("+strings.Join(placeholders, ",")+")")
	}
	if email := strings.ToLower(strings.TrimSpace(filter.Email)); email != "" {
		args = append(args, "%"+email+"%")
		clauses = append(clauses, "lower(email) LIKE ?")
	}
	if filter.Since != nil {
		args = append(args, filter.Since.UTC().Format("2006-01-02T15:04:05.000Z"))
		clauses = append(clauses, "created_at >= ?")
	}
	if filter.Until != nil {
		args = append(args, filter.Until.UTC().Format("2006-01-02T15:04:05.000Z"))
		clauses = append(clauses, "created_at <= ?")
	}
	args = append(args, limit+1, offset)
	query := fmt.Sprintf(
		`SELECT id, organization_id, invitation_id, action,
            actor_user_id, actor_email, email, role, note, created_at
         FROM organization_invitation_events
         WHERE %s
         ORDER BY created_at DESC
         LIMIT ? OFFSET ?`,
		strings.Join(clauses, " AND "),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return InvitationAuditPage{}, fmt.Errorf("list invitation audit: %w", err)
	}
	defer rows.Close()
	var out []domain.InvitationAuditEvent
	for rows.Next() {
		var ev domain.InvitationAuditEvent
		var actorUser, actorEmail, note sql.NullString
		var createdAt time.Time
		if scanErr := rows.Scan(
			&ev.ID,
			&ev.OrganizationID,
			&ev.InvitationID,
			&ev.Action,
			&actorUser,
			&actorEmail,
			&ev.Email,
			&ev.Role,
			&note,
			&createdAt,
		); scanErr != nil {
			return InvitationAuditPage{}, fmt.Errorf("scan invitation audit: %w", scanErr)
		}
		if actorUser.Valid {
			ev.ActorUserID = actorUser.String
		}
		if actorEmail.Valid {
			ev.ActorEmail = actorEmail.String
		}
		if note.Valid {
			ev.Note = note.String
		}
		ev.CreatedAt = createdAt.UTC()
		out = append(out, ev)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return InvitationAuditPage{}, rowsErr
	}
	hasMore := false
	if len(out) > limit {
		out = out[:limit]
		hasMore = true
	}
	return InvitationAuditPage{Events: out, HasMore: hasMore}, nil
}

func scanSQLiteInvitation(scanner sqliteScanner) (domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	var invitedBy sql.NullString
	var acceptedAt sql.NullTime
	var acceptedBy sql.NullString
	err := scanner.Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&invitedBy,
		&inv.Status,
		&inv.ExpiresAt,
		&acceptedAt,
		&acceptedBy,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		return domain.OrganizationInvitation{}, err
	}
	if invitedBy.Valid {
		inv.InvitedByUserID = invitedBy.String
	}
	if acceptedAt.Valid {
		t := acceptedAt.Time.UTC()
		inv.AcceptedAt = &t
	}
	if acceptedBy.Valid {
		inv.AcceptedByUserID = acceptedBy.String
	}
	return inv, nil
}
