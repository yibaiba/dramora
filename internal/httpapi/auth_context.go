package httpapi

import (
	"net/http"

	"github.com/yibaiba/dramora/internal/service"
)

// isAuthContextPublicPath 列出真正不需要登录就能命中的入口；其他 /api/v1/auth/*
// 子路径（例如 /auth/me、/auth/sessions）依赖 auth context middleware 来注入身份。
func isAuthContextPublicPath(path string) bool {
	switch path {
	case "/api/v1/meta/capabilities",
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout":
		return true
	}
	return false
}

func authContextMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authService == nil {
				next.ServeHTTP(w, r)
				return
			}
			if isAuthContextPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			session, err := authService.CurrentSession(r.Context(), authorization)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired credentials")
				return
			}

			ctx := service.WithRequestAuthContext(r.Context(), service.RequestAuthContext{
				UserID:         session.User.ID,
				OrganizationID: session.OrganizationID,
				Role:           session.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
