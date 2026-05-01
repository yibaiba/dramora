package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestPrometheusMetricsExposesWorkerCounters(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		ProductionService: productionService,
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", ct)
	}
	body := resp.Body.String()
	for _, want := range []string{
		"# TYPE dramora_worker_org_unresolved_skips_total counter",
		`dramora_worker_org_unresolved_skips_total{kind="generation"} 0`,
		`dramora_worker_org_unresolved_skips_total{kind="export"} 0`,
		"# TYPE dramora_worker_last_skip_timestamp_seconds gauge",
		"dramora_worker_last_skip_timestamp_seconds 0",
		"# TYPE dramora_worker_last_skip_info gauge",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected /metrics body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestPrometheusMetricsIsPublicAndUnauthenticated(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := repo.NewMemoryIdentityRepository()
	authService := service.NewAuthService(identityRepo, "test-secret", nil)
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProductionService: productionService,
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /metrics to be public, got %d", resp.Code)
	}
}
