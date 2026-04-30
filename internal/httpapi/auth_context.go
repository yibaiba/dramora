package httpapi

import (
	"net/http"
	"strings"

	"github.com/yibaiba/dramora/internal/service"
)

func authContextMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authService == nil {
				next.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") || r.URL.Path == "/api/v1/meta/capabilities" {
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
