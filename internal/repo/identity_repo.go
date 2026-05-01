package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

type AuthIdentity struct {
	User           domain.User
	OrganizationID string
	Role           string
	PasswordHash   string
}

type CreateUserWithMembershipParams struct {
	UserID         string
	OrganizationID string
	Email          string
	DisplayName    string
	PasswordHash   string
	Role           string
}

type IdentityRepository interface {
	CreateUserWithMembership(ctx context.Context, params CreateUserWithMembershipParams) (AuthIdentity, error)
	GetAuthIdentityByEmail(ctx context.Context, email string) (AuthIdentity, error)
	GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error)

	CreateOrganization(ctx context.Context, params CreateOrganizationParams) error
	CreateInvitation(ctx context.Context, params CreateInvitationParams) (domain.OrganizationInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (domain.OrganizationInvitation, error)
	MarkInvitationAccepted(ctx context.Context, invitationID, userID string, acceptedAt time.Time) error
	ListOrganizationInvitations(ctx context.Context, organizationID string) ([]domain.OrganizationInvitation, error)
	RevokeInvitation(ctx context.Context, invitationID, organizationID string, revokedAt time.Time) error
}

type CreateOrganizationParams struct {
	OrganizationID string
	Name           string
}

type CreateInvitationParams struct {
	InvitationID    string
	OrganizationID  string
	Email           string
	Role            string
	Token           string
	InvitedByUserID string
	ExpiresAt       time.Time
}

type PostgresIdentityRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIdentityRepository(pool *pgxpool.Pool) *PostgresIdentityRepository {
	return &PostgresIdentityRepository{pool: pool}
}

func (r *PostgresIdentityRepository) CreateUserWithMembership(
	ctx context.Context,
	params CreateUserWithMembershipParams,
) (AuthIdentity, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, createUserSQL,
		params.UserID,
		params.Email,
		params.DisplayName,
		params.PasswordHash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return AuthIdentity{}, domain.ErrNotFound
		}
		return AuthIdentity{}, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(ctx, createOrganizationMemberSQL, params.OrganizationID, params.UserID, params.Role)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return AuthIdentity{}, domain.ErrNotFound
		}
		return AuthIdentity{}, fmt.Errorf("create organization member: %w", err)
	}

	identity, err := scanAuthIdentity(tx.QueryRow(ctx, getAuthIdentityByUserIDSQL, params.UserID))
	if err != nil {
		return AuthIdentity{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthIdentity{}, fmt.Errorf("commit create user: %w", err)
	}
	return identity, nil
}

func (r *PostgresIdentityRepository) GetAuthIdentityByEmail(ctx context.Context, email string) (AuthIdentity, error) {
	identity, err := scanAuthIdentity(r.pool.QueryRow(ctx, getAuthIdentityByEmailSQL, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func (r *PostgresIdentityRepository) GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error) {
	identity, err := scanAuthIdentity(r.pool.QueryRow(ctx, getAuthIdentityByUserIDSQL, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

type sqliteScanner interface {
	Scan(dest ...any) error
}

func scanAuthIdentity(scanner sqliteScanner) (AuthIdentity, error) {
	var identity AuthIdentity
	err := scanner.Scan(
		&identity.User.ID,
		&identity.User.Email,
		&identity.User.DisplayName,
		&identity.PasswordHash,
		&identity.OrganizationID,
		&identity.Role,
		&identity.User.CreatedAt,
		&identity.User.UpdatedAt,
	)
	return identity, err
}

func (r *PostgresIdentityRepository) CreateOrganization(ctx context.Context, params CreateOrganizationParams) error {
	if _, err := r.pool.Exec(ctx, createOrganizationSQL, params.OrganizationID, params.Name); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *PostgresIdentityRepository) CreateInvitation(
	ctx context.Context,
	params CreateInvitationParams,
) (domain.OrganizationInvitation, error) {
	var invitedBy any
	if params.InvitedByUserID != "" {
		invitedBy = params.InvitedByUserID
	}
	row := r.pool.QueryRow(ctx, createInvitationSQL,
		params.InvitationID,
		params.OrganizationID,
		params.Email,
		params.Role,
		params.Token,
		invitedBy,
		params.ExpiresAt,
	)
	inv, err := scanInvitation(row)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrNotFound
		}
		return domain.OrganizationInvitation{}, fmt.Errorf("create invitation: %w", err)
	}
	return inv, nil
}

func (r *PostgresIdentityRepository) GetInvitationByToken(ctx context.Context, token string) (domain.OrganizationInvitation, error) {
	inv, err := scanInvitation(r.pool.QueryRow(ctx, getInvitationByTokenSQL, token))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrganizationInvitation{}, domain.ErrNotFound
	}
	return inv, err
}

func (r *PostgresIdentityRepository) MarkInvitationAccepted(ctx context.Context, invitationID, userID string, acceptedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, markInvitationAcceptedSQL, invitationID, userID, acceptedAt)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) ListOrganizationInvitations(ctx context.Context, organizationID string) ([]domain.OrganizationInvitation, error) {
	rows, err := r.pool.Query(ctx, listInvitationsByOrgSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()
	var out []domain.OrganizationInvitation
	for rows.Next() {
		inv, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan invitation: %w", scanErr)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *PostgresIdentityRepository) RevokeInvitation(ctx context.Context, invitationID, organizationID string, revokedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, revokeInvitationSQL, invitationID, organizationID, revokedAt.UTC())
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanInvitation(scanner sqliteScanner) (domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	var invitedBy *string
	var acceptedAt *time.Time
	var acceptedBy *string
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
	if invitedBy != nil {
		inv.InvitedByUserID = *invitedBy
	}
	if acceptedAt != nil {
		t := acceptedAt.UTC()
		inv.AcceptedAt = &t
	}
	if acceptedBy != nil {
		inv.AcceptedByUserID = *acceptedBy
	}
	return inv, nil
}
