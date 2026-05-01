package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

const sqliteCreateRefreshTokenSQL = `
INSERT INTO auth_refresh_tokens (id, user_id, organization_id, role, token_hash, expires_at)
VALUES (?, ?, ?, ?, ?, ?)
`

const sqliteRefreshTokenSelect = `
SELECT id, user_id, organization_id, role, token_hash, created_at, expires_at, revoked_at, replaced_by_id
FROM auth_refresh_tokens
`

const sqliteGetRefreshTokenByHashSQL = sqliteRefreshTokenSelect + `
WHERE token_hash = ?
LIMIT 1
`

const sqliteRevokeRefreshTokenSQL = `
UPDATE auth_refresh_tokens
SET revoked_at = COALESCE(revoked_at, strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    replaced_by_id = COALESCE(?, replaced_by_id)
WHERE id = ?
`

type SQLiteRefreshTokenRepository struct {
	db *sql.DB
}

func NewSQLiteRefreshTokenRepository(db *sql.DB) *SQLiteRefreshTokenRepository {
	return &SQLiteRefreshTokenRepository{db: db}
}

func (r *SQLiteRefreshTokenRepository) Create(ctx context.Context, params CreateRefreshTokenParams) (RefreshTokenRecord, error) {
	if _, err := r.db.ExecContext(ctx, sqliteCreateRefreshTokenSQL,
		params.ID, params.UserID, params.OrganizationID, params.Role,
		params.TokenHash, params.ExpiresAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		if isSQLiteUniqueViolation(err) {
			return RefreshTokenRecord{}, domain.ErrInvalidInput
		}
		if isSQLiteFKViolation(err) {
			return RefreshTokenRecord{}, domain.ErrNotFound
		}
		return RefreshTokenRecord{}, fmt.Errorf("create refresh token: %w", err)
	}
	rec, err := scanSQLiteRefreshTokenRow(r.db.QueryRowContext(ctx, sqliteGetRefreshTokenByHashSQL, params.TokenHash))
	if err != nil {
		return RefreshTokenRecord{}, err
	}
	return rec, nil
}

func (r *SQLiteRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	rec, err := scanSQLiteRefreshTokenRow(r.db.QueryRowContext(ctx, sqliteGetRefreshTokenByHashSQL, tokenHash))
	if err == sql.ErrNoRows {
		return RefreshTokenRecord{}, domain.ErrNotFound
	}
	return rec, err
}

func (r *SQLiteRefreshTokenRepository) Revoke(ctx context.Context, id string, replacedByID *string) error {
	res, err := r.db.ExecContext(ctx, sqliteRevokeRefreshTokenSQL, replacedByID, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke refresh token rows: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanSQLiteRefreshTokenRow(row *sql.Row) (RefreshTokenRecord, error) {
	var (
		rec          RefreshTokenRecord
		createdAt    string
		expiresAt    string
		revokedAt    sql.NullString
		replacedByID sql.NullString
	)
	err := row.Scan(
		&rec.ID, &rec.UserID, &rec.OrganizationID, &rec.Role, &rec.TokenHash,
		&createdAt, &expiresAt, &revokedAt, &replacedByID,
	)
	if err == sql.ErrNoRows {
		return RefreshTokenRecord{}, sql.ErrNoRows
	}
	if err != nil {
		return RefreshTokenRecord{}, fmt.Errorf("scan refresh token: %w", err)
	}
	if rec.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return RefreshTokenRecord{}, err
	}
	if rec.ExpiresAt, err = parseSQLiteTime(expiresAt); err != nil {
		return RefreshTokenRecord{}, err
	}
	if revokedAt.Valid {
		t, err := parseSQLiteTime(revokedAt.String)
		if err != nil {
			return RefreshTokenRecord{}, err
		}
		rec.RevokedAt = &t
	}
	if replacedByID.Valid {
		v := replacedByID.String
		rec.ReplacedByID = &v
	}
	return rec, nil
}

func parseSQLiteTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse sqlite time: unknown layout for %q", value)
}
