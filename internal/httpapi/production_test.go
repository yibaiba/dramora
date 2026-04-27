package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yibaiba/dramora/internal/jobs"
)

func TestProductionRoutesReturnEmptyOrNotFound(t *testing.T) {
	t.Parallel()

	router := testRouter()

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/generation-jobs", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflow-runs/missing", nil)
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", getResp.Code, getResp.Body.String())
	}
}

func TestStartStoryAnalysisRoute(t *testing.T) {
	t.Parallel()

	router := testRouter()

	projectResp := httptest.NewRecorder()
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body(`{"name":"Workflow Project"}`))
	router.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("expected project 201, got %d: %s", projectResp.Code, projectResp.Body.String())
	}

	var createdProject struct {
		Project projectResponse `json:"project"`
	}
	decodeBody(t, projectResp, &createdProject)

	episodeResp := httptest.NewRecorder()
	episodeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+createdProject.Project.ID+"/episodes",
		body(`{"title":"Pilot"}`),
	)
	router.ServeHTTP(episodeResp, episodeReq)
	if episodeResp.Code != http.StatusCreated {
		t.Fatalf("expected episode 201, got %d: %s", episodeResp.Code, episodeResp.Body.String())
	}

	var createdEpisode struct {
		Episode episodeResponse `json:"episode"`
	}
	decodeBody(t, episodeResp, &createdEpisode)

	startResp := httptest.NewRecorder()
	startReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+createdEpisode.Episode.ID+"/story-analysis/start",
		nil,
	)
	router.ServeHTTP(startResp, startReq)
	if startResp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", startResp.Code, startResp.Body.String())
	}

	var payload struct {
		WorkflowRun   workflowRunResponse   `json:"workflow_run"`
		GenerationJob generationJobResponse `json:"generation_job"`
	}
	decodeBody(t, startResp, &payload)
	if payload.WorkflowRun.Status != "running" {
		t.Fatalf("expected running workflow, got %q", payload.WorkflowRun.Status)
	}
	if payload.GenerationJob.Status != "queued" {
		t.Fatalf("expected queued job, got %q", payload.GenerationJob.Status)
	}
}

func TestSaveEpisodeTimelineRoute(t *testing.T) {
	t.Parallel()

	router := testRouter()
	episode := createTestEpisode(t, router)

	saveResp := httptest.NewRecorder()
	saveReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+episode.ID+"/timeline",
		body(`{"duration_ms":15000}`),
	)
	router.ServeHTTP(saveResp, saveReq)
	if saveResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", saveResp.Code, saveResp.Body.String())
	}

	var payload struct {
		Timeline timelineResponse `json:"timeline"`
	}
	decodeBody(t, saveResp, &payload)
	if payload.Timeline.DurationMS != 15000 {
		t.Fatalf("expected duration 15000, got %d", payload.Timeline.DurationMS)
	}

	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/timeline", nil)
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getResp.Code, getResp.Body.String())
	}
}

func TestStoryAnalysisReadRoutes(t *testing.T) {
	t.Parallel()

	router, productionService := testRouterWithProductionService()
	episode := createTestEpisode(t, router)

	startResp := httptest.NewRecorder()
	startReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+episode.ID+"/story-analysis/start",
		nil,
	)
	router.ServeHTTP(startResp, startReq)
	if startResp.Code != http.StatusAccepted {
		t.Fatalf("expected start 202, got %d: %s", startResp.Code, startResp.Body.String())
	}

	if _, err := productionService.ProcessQueuedGenerationJobs(
		startReq.Context(),
		jobs.DefaultExecutionLimit,
	); err != nil {
		t.Fatalf("process story analysis job: %v", err)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/story-analyses", nil)
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	var payload struct {
		StoryAnalyses []storyAnalysisResponse `json:"story_analyses"`
	}
	decodeBody(t, listResp, &payload)
	if len(payload.StoryAnalyses) != 1 {
		t.Fatalf("expected 1 completed analysis, got %d", len(payload.StoryAnalyses))
	}
	if got := payload.StoryAnalyses[0].CharacterSeeds; len(got) == 0 {
		t.Fatalf("expected character seeds, got %+v", payload.StoryAnalyses[0])
	}

	detailResp := httptest.NewRecorder()
	detailReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/story-analyses/"+payload.StoryAnalyses[0].ID,
		nil,
	)
	router.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", detailResp.Code, detailResp.Body.String())
	}
}

func TestCoreProductionMapStoryboardTimelineAndExportRoutes(t *testing.T) {
	t.Parallel()

	router, productionService := testRouterWithProductionService()
	episode := createTestEpisode(t, router)
	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/story-analysis/start", nil)
	startResp := httptest.NewRecorder()
	router.ServeHTTP(startResp, startReq)
	if startResp.Code != http.StatusAccepted {
		t.Fatalf("expected start 202, got %d: %s", startResp.Code, startResp.Body.String())
	}
	if _, err := productionService.ProcessQueuedGenerationJobs(startReq.Context(), jobs.DefaultExecutionLimit); err != nil {
		t.Fatalf("process story analysis job: %v", err)
	}

	storyMapResp := httptest.NewRecorder()
	storyMapReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/story-map:seed", nil)
	router.ServeHTTP(storyMapResp, storyMapReq)
	if storyMapResp.Code != http.StatusCreated {
		t.Fatalf("expected story map 201, got %d: %s", storyMapResp.Code, storyMapResp.Body.String())
	}
	var storyMapPayload struct {
		StoryMap storyMapResponse `json:"story_map"`
	}
	decodeBody(t, storyMapResp, &storyMapPayload)
	if len(storyMapPayload.StoryMap.Characters) == 0 || len(storyMapPayload.StoryMap.Scenes) == 0 {
		t.Fatalf("expected seeded story map, got %+v", storyMapPayload.StoryMap)
	}

	shotResp := httptest.NewRecorder()
	shotReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/storyboard-shots:seed", nil)
	router.ServeHTTP(shotResp, shotReq)
	if shotResp.Code != http.StatusCreated {
		t.Fatalf("expected storyboard 201, got %d: %s", shotResp.Code, shotResp.Body.String())
	}

	timelineResp := httptest.NewRecorder()
	timelineReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+episode.ID+"/timeline",
		body(`{"duration_ms":9000,"tracks":[{"kind":"video","name":"Video","position":1,"clips":[{"kind":"shot","start_ms":0,"duration_ms":3000}]}]}`),
	)
	router.ServeHTTP(timelineResp, timelineReq)
	if timelineResp.Code != http.StatusOK {
		t.Fatalf("expected timeline 200, got %d: %s", timelineResp.Code, timelineResp.Body.String())
	}

	exportResp := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/exports", nil)
	router.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusAccepted {
		t.Fatalf("expected export 202, got %d: %s", exportResp.Code, exportResp.Body.String())
	}
}

func createTestEpisode(t *testing.T, router http.Handler) episodeResponse {
	t.Helper()

	projectResp := httptest.NewRecorder()
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body(`{"name":"Timeline Project"}`))
	router.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("expected project 201, got %d: %s", projectResp.Code, projectResp.Body.String())
	}

	var createdProject struct {
		Project projectResponse `json:"project"`
	}
	decodeBody(t, projectResp, &createdProject)

	episodeResp := httptest.NewRecorder()
	episodeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+createdProject.Project.ID+"/episodes",
		body(`{"title":"Timeline Episode"}`),
	)
	router.ServeHTTP(episodeResp, episodeReq)
	if episodeResp.Code != http.StatusCreated {
		t.Fatalf("expected episode 201, got %d: %s", episodeResp.Code, episodeResp.Body.String())
	}

	var createdEpisode struct {
		Episode episodeResponse `json:"episode"`
	}
	decodeBody(t, episodeResp, &createdEpisode)
	return createdEpisode.Episode
}

func body(value string) *bytes.Buffer {
	return bytes.NewBufferString(value)
}
