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

	identity, err := scanAuthIdentity(tx.QueryRowContext(ctx, sqliteGetAuthIdentityByUserIDSQL, params.UserID))
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
	identity, err := scanAuthIdentity(r.db.QueryRowContext(ctx, sqliteGetAuthIdentityByEmailSQL, email))
	if err == sql.ErrNoRows {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func (r *SQLiteIdentityRepository) GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error) {
	identity, err := scanAuthIdentity(r.db.QueryRowContext(ctx, sqliteGetAuthIdentityByUserIDSQL, userID))
	if err == sql.ErrNoRows {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
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
