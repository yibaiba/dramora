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

const pgCreateRefreshTokenSQL = `
INSERT INTO auth_refresh_tokens (id, user_id, organization_id, role, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, organization_id, role, token_hash, created_at, expires_at, revoked_at, replaced_by_id
`

const pgGetRefreshTokenByHashSQL = `
SELECT id, user_id, organization_id, role, token_hash, created_at, expires_at, revoked_at, replaced_by_id
FROM auth_refresh_tokens
WHERE token_hash = $1
LIMIT 1
`

const pgRevokeRefreshTokenSQL = `
UPDATE auth_refresh_tokens
SET revoked_at = COALESCE(revoked_at, NOW()),
    replaced_by_id = COALESCE($2, replaced_by_id)
WHERE id = $1
`

type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, params CreateRefreshTokenParams) (RefreshTokenRecord, error) {
	rec, err := scanPGRefreshTokenRow(r.pool.QueryRow(ctx, pgCreateRefreshTokenSQL,
		params.ID, params.UserID, params.OrganizationID, params.Role,
		params.TokenHash, params.ExpiresAt.UTC(),
	))
	if err != nil {
		if isUniqueViolation(err) {
			return RefreshTokenRecord{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return RefreshTokenRecord{}, domain.ErrNotFound
		}
		return RefreshTokenRecord{}, fmt.Errorf("create refresh token: %w", err)
	}
	return rec, nil
}

func (r *PostgresRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	rec, err := scanPGRefreshTokenRow(r.pool.QueryRow(ctx, pgGetRefreshTokenByHashSQL, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshTokenRecord{}, domain.ErrNotFound
	}
	return rec, err
}

func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id string, replacedByID *string) error {
	cmd, err := r.pool.Exec(ctx, pgRevokeRefreshTokenSQL, id, replacedByID)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type pgRowScanner interface {
	Scan(dest ...any) error
}

func scanPGRefreshTokenRow(row pgRowScanner) (RefreshTokenRecord, error) {
	var (
		rec          RefreshTokenRecord
		revokedAt    *time.Time
		replacedByID *string
	)
	if err := row.Scan(
		&rec.ID, &rec.UserID, &rec.OrganizationID, &rec.Role, &rec.TokenHash,
		&rec.CreatedAt, &rec.ExpiresAt, &revokedAt, &replacedByID,
	); err != nil {
		return RefreshTokenRecord{}, err
	}
	if revokedAt != nil {
		t := revokedAt.UTC()
		rec.RevokedAt = &t
	}
	rec.CreatedAt = rec.CreatedAt.UTC()
	rec.ExpiresAt = rec.ExpiresAt.UTC()
	rec.ReplacedByID = replacedByID
	return rec, nil
}
