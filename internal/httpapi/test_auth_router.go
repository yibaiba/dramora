package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/yibaiba/dramora/internal/service"
)

type authenticatedTestRouter struct {
	authorization string
	handler       http.Handler
}

func newAuthenticatedTestRouter(handler http.Handler, authService *service.AuthService) http.Handler {
	session, err := authService.Register(context.Background(), service.RegisterInput{
		Email:       "test-director@example.com",
		DisplayName: "Test Director",
		Password:    "strongpass",
	})
	if err != nil {
		panic(err)
	}
	return authenticatedTestRouter{
		authorization: "Bearer " + session.Token,
		handler:       handler,
	}
}

func (r authenticatedTestRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/api/v1/") &&
		!strings.HasPrefix(req.URL.Path, "/api/v1/auth/") &&
		req.URL.Path != "/api/v1/meta/capabilities" &&
		req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", r.authorization)
	}
	r.handler.ServeHTTP(w, req)
}
