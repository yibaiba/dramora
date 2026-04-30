package repo

import (
	"context"
	"database/sql"
	"fmt"
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
