package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(recorder, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status(),
				"bytes", recorder.BytesWritten(),
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
