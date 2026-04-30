package service

import "context"

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
