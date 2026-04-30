package repo

import (
	"context"
	"errors"
	"fmt"

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
