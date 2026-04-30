package service

import (
	"context"
	"strings"
)

const (
	// RoleSystem 标识来自后台 worker / 内部任务的系统级身份。
	// 该上下文绕过组织归属过滤，仅供 inline worker / job runner 等
	// 没有真实用户会话但可信的执行环境注入。
	RoleSystem = "system"
)

type requestAuthContextKey struct{}

type RequestAuthContext struct {
	UserID         string
	OrganizationID string
	Role           string
}

func WithRequestAuthContext(ctx context.Context, auth RequestAuthContext) context.Context {
	return context.WithValue(ctx, requestAuthContextKey{}, auth)
}

func RequestAuthFromContext(ctx context.Context) (RequestAuthContext, bool) {
	auth, ok := ctx.Value(requestAuthContextKey{}).(RequestAuthContext)
	return auth, ok
}

// WithSystemAuthContext 为后台任务注入系统身份。
func WithSystemAuthContext(ctx context.Context) context.Context {
	return WithRequestAuthContext(ctx, RequestAuthContext{Role: RoleSystem})
}

// IsSystemAuthContext 判断当前 ctx 是否带有 system 角色。
func IsSystemAuthContext(ctx context.Context) bool {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(auth.Role), RoleSystem)
}
