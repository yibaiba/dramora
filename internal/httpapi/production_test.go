package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
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
