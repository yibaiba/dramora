package repo

import (
	"context"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

// RefreshTokenRecord 是 auth_refresh_tokens 表的领域投影。
type RefreshTokenRecord struct {
	ID             string
	UserID         string
	OrganizationID string
	Role           string
	TokenHash      string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	ReplacedByID   *string
}

// CreateRefreshTokenParams 用于插入一条新的 refresh token 行。
type CreateRefreshTokenParams struct {
	ID             string
	UserID         string
	OrganizationID string
	Role           string
	TokenHash      string
	ExpiresAt      time.Time
}

// RefreshTokenRepository 管理 refresh token 的存储与轮换。
//
// 实现需保证：
//   - GetByHash 返回未吊销且未过期的有效 token，否则 domain.ErrNotFound 或 domain.ErrInvalidInput。
//   - Revoke 是幂等操作；吊销已吊销的 token 不应报错。
type RefreshTokenRepository interface {
	Create(ctx context.Context, params CreateRefreshTokenParams) (RefreshTokenRecord, error)
	GetByHash(ctx context.Context, tokenHash string) (RefreshTokenRecord, error)
	Revoke(ctx context.Context, id string, replacedByID *string) error
}

// 显式声明 sentinel，便于上层 errors.Is 判断。
var (
	_ = domain.ErrNotFound
)
