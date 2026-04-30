package httpapi

import (
	"net/http"

	"github.com/yibaiba/dramora/internal/service"
)

func requireRole(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth, ok := service.RequestAuthFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			if _, allowed := allowedSet[auth.Role]; !allowed {
				writeError(w, http.StatusForbidden, "forbidden", "insufficient role for this resource")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
