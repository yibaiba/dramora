package repo

import (
	"context"
	"database/sql"
	"fmt"

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
