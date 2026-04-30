package service

import (
	"context"
)

const (
	// RoleWorker 标识 worker 处理具体 job 时按 job 所属组织注入的上下文。
	// Worker 上下文必须显式携带真实 OrganizationID，与普通用户身份一样
	// 走 authorize 检查；不再提供任何"系统级 bypass"。
	RoleWorker = "worker"
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
