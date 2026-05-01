package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestProjectAndEpisodeRoutes(t *testing.T) {
	t.Parallel()

	router := testRouter()

	projectBody := bytes.NewBufferString(`{"name":"漫幕","description":"AI manju studio"}`)
	projectResp := httptest.NewRecorder()
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	router.ServeHTTP(projectResp, projectReq)

	if projectResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", projectResp.Code, projectResp.Body.String())
	}

	var createdProject struct {
		Project projectResponse `json:"project"`
	}
	decodeBody(t, projectResp, &createdProject)
	if createdProject.Project.Name != "漫幕" {
		t.Fatalf("expected project name, got %q", createdProject.Project.Name)
	}

	episodeURL := "/api/v1/projects/" + createdProject.Project.ID + "/episodes"
	episodeResp := httptest.NewRecorder()
	episodeReq := httptest.NewRequest(http.MethodPost, episodeURL, bytes.NewBufferString(`{"title":"第一集"}`))
	router.ServeHTTP(episodeResp, episodeReq)

	if episodeResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", episodeResp.Code, episodeResp.Body.String())
	}

	var createdEpisode struct {
		Episode episodeResponse `json:"episode"`
	}
	decodeBody(t, episodeResp, &createdEpisode)
	if createdEpisode.Episode.Number != 1 {
		t.Fatalf("expected episode number 1, got %d", createdEpisode.Episode.Number)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, episodeURL, nil)
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}
}

func TestProjectRouteValidation(t *testing.T) {
	t.Parallel()

	router := testRouter()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{"name":" "}`))
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}

	var payload errorResponse
	decodeBody(t, resp, &payload)
	if payload.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %q", payload.Error.Code)
	}
}

func testRouter() http.Handler {
	router, _ := testRouterWithProductionService()
	return router
}

func testRouterWithProductionService() (http.Handler, *service.ProductionService) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	projectService := service.NewProjectService(repo.NewMemoryProjectRepository())
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret")
	authService.SetRefreshTokenRepository(repo.NewMemoryRefreshTokenRepository())
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	productionService.SetProjectService(projectService)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProjectService:    projectService,
		ProductionService: productionService,
	})
	return newAuthenticatedTestRouter(router, authService), productionService
}

func decodeBody(t *testing.T, resp *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
