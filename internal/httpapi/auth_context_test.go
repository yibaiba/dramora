package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func TestProjectRoutesUseOrganizationFromJWTContext(t *testing.T) {
	t.Parallel()

	const defaultOrgID = "00000000-0000-0000-0000-000000000001"
	const authOrgID = "00000000-0000-0000-0000-000000000099"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := service.NewProjectService(projectRepo)
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)
	router := NewRouter(RouterConfig{
		Logger:         logger,
		Version:        "test",
		AuthService:    authService,
		ProjectService: projectService,
	})

	if _, err := projectService.CreateProject(context.Background(), service.CreateProjectInput{
		Name: "Default Org Project",
	}); err != nil {
		t.Fatalf("seed default org project: %v", err)
	}

	registerResp := httptest.NewRecorder()
	registerReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		body(`{"email":"orgb@example.com","display_name":"Org B","password":"strongpass"}`),
	)
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d: %s", registerResp.Code, registerResp.Body.String())
	}

	var registerPayload struct {
		Session authSessionResponse `json:"session"`
	}
	decodeBody(t, registerResp, &registerPayload)

	authProjectResp := httptest.NewRecorder()
	authProjectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body(`{"name":"Auth Org Project"}`))
	authProjectReq.Header.Set("Authorization", "Bearer "+registerPayload.Session.Token)
	router.ServeHTTP(authProjectResp, authProjectReq)
	if authProjectResp.Code != http.StatusCreated {
		t.Fatalf("expected auth project 201, got %d: %s", authProjectResp.Code, authProjectResp.Body.String())
	}

	authListResp := httptest.NewRecorder()
	authListReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	authListReq.Header.Set("Authorization", "Bearer "+registerPayload.Session.Token)
	router.ServeHTTP(authListResp, authListReq)
	if authListResp.Code != http.StatusOK {
		t.Fatalf("expected auth list 200, got %d: %s", authListResp.Code, authListResp.Body.String())
	}

	var authListPayload struct {
		Projects []projectResponse `json:"projects"`
	}
	decodeBody(t, authListResp, &authListPayload)
	if len(authListPayload.Projects) != 1 || authListPayload.Projects[0].Name != "Auth Org Project" {
		t.Fatalf("expected auth org project only, got %+v", authListPayload.Projects)
	}

	if authListPayload.Projects[0].Name == "Default Org Project" {
		t.Fatalf("did not expect auth org to see default org project: %+v", authListPayload.Projects)
	}
}

func TestProjectRoutesRejectInvalidBearerToken(t *testing.T) {
	t.Parallel()

	router := testRouter()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestProjectRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(RouterConfig{
		Logger:         logger,
		Version:        "test",
		AuthService:    service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil),
		ProjectService: service.NewProjectService(repo.NewMemoryProjectRepository()),
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestProductionRoutesRespectOrganizationContext(t *testing.T) {
	t.Parallel()

	const defaultOrgID = "00000000-0000-0000-0000-000000000001"
	const authOrgID = "00000000-0000-0000-0000-000000000099"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := service.NewProjectService(projectRepo)
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	productionService.SetProjectService(projectService)
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProjectService:    projectService,
		ProductionService: productionService,
	})

	project, err := projectService.CreateProject(context.Background(), service.CreateProjectInput{Name: "Scoped Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(context.Background(), service.CreateEpisodeInput{
		ProjectID: project.ID,
		Title:     "Scoped Episode",
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	started, err := productionService.StartStoryAnalysis(context.Background(), episode)
	if err != nil {
		t.Fatalf("start story analysis: %v", err)
	}
	if _, err := productionService.SaveEpisodeTimeline(context.Background(), service.SaveTimelineInput{
		EpisodeID:  episode.ID,
		DurationMS: 12_000,
	}); err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	export, err := productionService.StartEpisodeExport(context.Background(), episode.ID)
	if err != nil {
		t.Fatalf("start export: %v", err)
	}

	session, err := authService.Register(context.Background(), service.RegisterInput{
		Email:       "other-org@example.com",
		DisplayName: "Other Org",
		Password:    "strongpass",
	})
	if err != nil {
		t.Fatalf("register auth session: %v", err)
	}
	authHeader := "Bearer " + session.Token

	jobsResp := httptest.NewRecorder()
	jobsReq := httptest.NewRequest(http.MethodGet, "/api/v1/generation-jobs", nil)
	jobsReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(jobsResp, jobsReq)
	if jobsResp.Code != http.StatusOK {
		t.Fatalf("expected jobs 200, got %d: %s", jobsResp.Code, jobsResp.Body.String())
	}
	var jobsPayload struct {
		GenerationJobs []generationJobResponse `json:"generation_jobs"`
	}
	decodeBody(t, jobsResp, &jobsPayload)
	if len(jobsPayload.GenerationJobs) != 0 {
		t.Fatalf("expected no generation jobs for other org, got %+v", jobsPayload.GenerationJobs)
	}

	workflowResp := httptest.NewRecorder()
	workflowReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflow-runs/"+started.WorkflowRun.ID, nil)
	workflowReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(workflowResp, workflowReq)
	if workflowResp.Code != http.StatusNotFound {
		t.Fatalf("expected workflow 404, got %d: %s", workflowResp.Code, workflowResp.Body.String())
	}

	jobResp := httptest.NewRecorder()
	jobReq := httptest.NewRequest(http.MethodGet, "/api/v1/generation-jobs/"+started.GenerationJob.ID, nil)
	jobReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(jobResp, jobReq)
	if jobResp.Code != http.StatusNotFound {
		t.Fatalf("expected generation job 404, got %d: %s", jobResp.Code, jobResp.Body.String())
	}

	exportResp := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+export.ID, nil)
	exportReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusNotFound {
		t.Fatalf("expected export 404, got %d: %s", exportResp.Code, exportResp.Body.String())
	}
}

func TestGlobalResourceWriteRoutesRespectOrganizationContext(t *testing.T) {
	t.Parallel()

	const defaultOrgID = "00000000-0000-0000-0000-000000000001"
	const authOrgID = "00000000-0000-0000-0000-000000000099"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := service.NewProjectService(projectRepo)
	productionService := service.NewProductionService(repo.NewMemoryProductionRepository(), nil)
	productionService.SetProjectService(projectService)
	authService := service.NewAuthService(repo.NewMemoryIdentityRepository(), "test-secret", nil)
	router := NewRouter(RouterConfig{
		Logger:            logger,
		Version:           "test",
		AuthService:       authService,
		ProjectService:    projectService,
		ProductionService: productionService,
	})

	seedCtx := service.WithRequestAuthContext(context.Background(), service.RequestAuthContext{
		OrganizationID: defaultOrgID,
		Role:           "owner",
	})

	project, err := projectService.CreateProject(seedCtx, service.CreateProjectInput{Name: "Protected Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(seedCtx, service.CreateEpisodeInput{
		ProjectID: project.ID,
		Title:     "Protected Episode",
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := productionService.StartStoryAnalysis(seedCtx, episode); err != nil {
		t.Fatalf("start story analysis: %v", err)
	}
	if _, err := productionService.ProcessQueuedGenerationJobs(context.Background(), jobs.DefaultExecutionLimit); err != nil {
		t.Fatalf("process generation jobs: %v", err)
	}
	storyMap, err := productionService.SeedStoryMap(seedCtx, episode)
	if err != nil {
		t.Fatalf("seed story map: %v", err)
	}
	assets, err := productionService.SeedEpisodeAssets(seedCtx, episode)
	if err != nil {
		t.Fatalf("seed assets: %v", err)
	}
	shots, err := productionService.SeedStoryboardShots(seedCtx, episode)
	if err != nil {
		t.Fatalf("seed shots: %v", err)
	}
	if _, err := productionService.SaveEpisodeTimeline(seedCtx, service.SaveTimelineInput{
		EpisodeID:  episode.ID,
		DurationMS: 12_000,
	}); err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	gates, err := productionService.SeedEpisodeApprovalGates(seedCtx, episode)
	if err != nil {
		t.Fatalf("seed approval gates: %v", err)
	}
	if len(storyMap.Characters) == 0 || len(assets) == 0 || len(shots) == 0 || len(gates) == 0 {
		t.Fatalf("expected seeded resources, got characters=%d assets=%d shots=%d gates=%d", len(storyMap.Characters), len(assets), len(shots), len(gates))
	}

	session, err := authService.Register(context.Background(), service.RegisterInput{
		Email:       "other-resource-org@example.com",
		DisplayName: "Other Resource Org",
		Password:    "strongpass",
	})
	if err != nil {
		t.Fatalf("register auth session: %v", err)
	}
	authHeader := "Bearer " + session.Token

	lockResp := httptest.NewRecorder()
	lockReq := httptest.NewRequest(http.MethodPost, "/api/v1/assets/"+assets[0].ID+":lock", nil)
	lockReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(lockResp, lockReq)
	if lockResp.Code != http.StatusNotFound {
		t.Fatalf("expected lock asset 404, got %d: %s", lockResp.Code, lockResp.Body.String())
	}

	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shots[0].ID+":update",
		body(`{"title":"Alt","description":"Nope","prompt":"No access","duration_ms":1200}`),
	)
	updateReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusNotFound {
		t.Fatalf("expected update shot 404, got %d: %s", updateResp.Code, updateResp.Body.String())
	}

	promptResp := httptest.NewRecorder()
	promptReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shots[0].ID+"/prompt-pack:generate",
		nil,
	)
	promptReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(promptResp, promptReq)
	if promptResp.Code != http.StatusNotFound {
		t.Fatalf("expected generate prompt pack 404, got %d: %s", promptResp.Code, promptResp.Body.String())
	}

	gateResp := httptest.NewRecorder()
	gateReq := httptest.NewRequest(http.MethodPost, "/api/v1/approval-gates/"+gates[0].ID+":approve", body(`{}`))
	gateReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(gateResp, gateReq)
	if gateResp.Code != http.StatusNotFound {
		t.Fatalf("expected approval gate 404, got %d: %s", gateResp.Code, gateResp.Body.String())
	}

	characterResp := httptest.NewRecorder()
	characterReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/story-map-characters/"+storyMap.Characters[0].ID+"/character-bible:save",
		body(`{"character_bible":{"anchor":"hero","palette":{"skin":"fair","hair":"black","accent":"gold","eyes":"amber","costume":"white"},"expressions":["calm"],"reference_angles":["front"],"reference_assets":[],"wardrobe":"robe","notes":"no access"}}`),
	)
	characterReq.Header.Set("Authorization", authHeader)
	router.ServeHTTP(characterResp, characterReq)
	if characterResp.Code != http.StatusNotFound {
		t.Fatalf("expected character bible 404, got %d: %s", characterResp.Code, characterResp.Body.String())
	}
}
