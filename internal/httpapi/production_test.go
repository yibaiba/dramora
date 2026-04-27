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

	invalidResp := httptest.NewRecorder()
	invalidReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+episode.ID+"/timeline",
		body(`{"duration_ms":1000,"tracks":[{"kind":"video","name":"Video","position":1,"clips":[{"kind":"shot","start_ms":900,"duration_ms":200}]}]}`),
	)
	router.ServeHTTP(invalidResp, invalidReq)
	if invalidResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid timeline 400, got %d: %s", invalidResp.Code, invalidResp.Body.String())
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

	assetsResp := httptest.NewRecorder()
	assetsReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/assets:seed", nil)
	router.ServeHTTP(assetsResp, assetsReq)
	if assetsResp.Code != http.StatusCreated {
		t.Fatalf("expected assets 201, got %d: %s", assetsResp.Code, assetsResp.Body.String())
	}
	var assetsPayload struct {
		Assets []assetResponse `json:"assets"`
	}
	decodeBody(t, assetsResp, &assetsPayload)
	if len(assetsPayload.Assets) == 0 {
		t.Fatal("expected seeded asset candidates")
	}

	lockResp := httptest.NewRecorder()
	lockReq := httptest.NewRequest(http.MethodPost, "/api/v1/assets/"+assetsPayload.Assets[0].ID+":lock", nil)
	router.ServeHTTP(lockResp, lockReq)
	if lockResp.Code != http.StatusOK {
		t.Fatalf("expected lock 200, got %d: %s", lockResp.Code, lockResp.Body.String())
	}
	var lockedPayload struct {
		Asset assetResponse `json:"asset"`
	}
	decodeBody(t, lockResp, &lockedPayload)
	if lockedPayload.Asset.Status != "ready" {
		t.Fatalf("expected locked asset ready, got %q", lockedPayload.Asset.Status)
	}
	if len(assetsPayload.Assets) > 1 {
		secondLockResp := httptest.NewRecorder()
		secondLockReq := httptest.NewRequest(http.MethodPost, "/api/v1/assets/"+assetsPayload.Assets[1].ID+":lock", nil)
		router.ServeHTTP(secondLockResp, secondLockReq)
		if secondLockResp.Code != http.StatusOK {
			t.Fatalf("expected second lock 200, got %d: %s", secondLockResp.Code, secondLockResp.Body.String())
		}
	}

	shotResp := httptest.NewRecorder()
	shotReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/storyboard-shots:seed", nil)
	router.ServeHTTP(shotResp, shotReq)
	if shotResp.Code != http.StatusCreated {
		t.Fatalf("expected storyboard 201, got %d: %s", shotResp.Code, shotResp.Body.String())
	}
	var shotPayload struct {
		StoryboardShots []storyboardShotResponse `json:"storyboard_shots"`
	}
	decodeBody(t, shotResp, &shotPayload)
	if len(shotPayload.StoryboardShots) == 0 {
		t.Fatal("expected seeded storyboard shots")
	}
	updateShotResp := httptest.NewRecorder()
	updateShotReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+":update",
		body(`{"title":"开场云海重构","description":"强化主角登场动机","prompt":"电影感云海，少年立于天门之前","duration_ms":4200}`),
	)
	router.ServeHTTP(updateShotResp, updateShotReq)
	if updateShotResp.Code != http.StatusOK {
		t.Fatalf("expected shot update 200, got %d: %s", updateShotResp.Code, updateShotResp.Body.String())
	}
	var updatedShotPayload struct {
		StoryboardShot storyboardShotResponse `json:"storyboard_shot"`
	}
	decodeBody(t, updateShotResp, &updatedShotPayload)
	if updatedShotPayload.StoryboardShot.Title != "开场云海重构" || updatedShotPayload.StoryboardShot.DurationMS != 4200 {
		t.Fatalf("expected updated shot fields, got %+v", updatedShotPayload.StoryboardShot)
	}

	promptResp := httptest.NewRecorder()
	promptReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+"/prompt-pack:generate",
		nil,
	)
	router.ServeHTTP(promptResp, promptReq)
	if promptResp.Code != http.StatusCreated {
		t.Fatalf("expected prompt pack 201, got %d: %s", promptResp.Code, promptResp.Body.String())
	}
	var promptPayload struct {
		PromptPack shotPromptPackResponse `json:"prompt_pack"`
	}
	decodeBody(t, promptResp, &promptPayload)
	if promptPayload.PromptPack.Preset != "sd2_fast" {
		t.Fatalf("expected sd2_fast preset, got %q", promptPayload.PromptPack.Preset)
	}
	if promptPayload.PromptPack.TaskType != "image_to_video" {
		t.Fatalf("expected image-to-video prompt pack, got %q", promptPayload.PromptPack.TaskType)
	}
	if len(promptPayload.PromptPack.ReferenceBindings) < 2 || promptPayload.PromptPack.ReferenceBindings[1].Token != "@image2" {
		t.Fatalf("expected image references including @image2, got %+v", promptPayload.PromptPack.ReferenceBindings)
	}
	savePromptResp := httptest.NewRecorder()
	savePromptReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+"/prompt-pack:save",
		body(`{"direct_prompt":"保留角色一致性，增加云海纵深和逆光轮廓"}`),
	)
	router.ServeHTTP(savePromptResp, savePromptReq)
	if savePromptResp.Code != http.StatusOK {
		t.Fatalf("expected prompt save 200, got %d: %s", savePromptResp.Code, savePromptResp.Body.String())
	}
	var savedPromptPayload struct {
		PromptPack shotPromptPackResponse `json:"prompt_pack"`
	}
	decodeBody(t, savePromptResp, &savedPromptPayload)
	if savedPromptPayload.PromptPack.DirectPrompt != "保留角色一致性，增加云海纵深和逆光轮廓" {
		t.Fatalf("expected saved direct prompt, got %q", savedPromptPayload.PromptPack.DirectPrompt)
	}
	videoResp := httptest.NewRecorder()
	videoReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+"/videos:generate",
		nil,
	)
	router.ServeHTTP(videoResp, videoReq)
	if videoResp.Code != http.StatusAccepted {
		t.Fatalf("expected video generation 202, got %d: %s", videoResp.Code, videoResp.Body.String())
	}
	var videoPayload struct {
		GenerationJob generationJobResponse `json:"generation_job"`
	}
	decodeBody(t, videoResp, &videoPayload)
	if videoPayload.GenerationJob.Provider != "seedance" {
		t.Fatalf("expected seedance provider, got %q", videoPayload.GenerationJob.Provider)
	}
	if videoPayload.GenerationJob.TaskType != "image_to_video" {
		t.Fatalf("expected image-to-video job, got %q", videoPayload.GenerationJob.TaskType)
	}
	duplicateVideoResp := httptest.NewRecorder()
	duplicateVideoReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+"/videos:generate",
		nil,
	)
	router.ServeHTTP(duplicateVideoResp, duplicateVideoReq)
	if duplicateVideoResp.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate video generation 202, got %d: %s", duplicateVideoResp.Code, duplicateVideoResp.Body.String())
	}
	var duplicateVideoPayload struct {
		GenerationJob generationJobResponse `json:"generation_job"`
	}
	decodeBody(t, duplicateVideoResp, &duplicateVideoPayload)
	if duplicateVideoPayload.GenerationJob.ID != videoPayload.GenerationJob.ID {
		t.Fatalf("expected idempotent video job %q, got %q", videoPayload.GenerationJob.ID, duplicateVideoPayload.GenerationJob.ID)
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

	gatesResp := httptest.NewRecorder()
	gatesReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/approval-gates:seed", nil)
	router.ServeHTTP(gatesResp, gatesReq)
	if gatesResp.Code != http.StatusCreated {
		t.Fatalf("expected approval gates 201, got %d: %s", gatesResp.Code, gatesResp.Body.String())
	}
	var gatesPayload struct {
		ApprovalGates []approvalGateResponse `json:"approval_gates"`
	}
	decodeBody(t, gatesResp, &gatesPayload)
	if len(gatesPayload.ApprovalGates) < 6 {
		t.Fatalf("expected seeded approval gates, got %+v", gatesPayload.ApprovalGates)
	}
	approveResp := httptest.NewRecorder()
	approveReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/approval-gates/"+gatesPayload.ApprovalGates[0].ID+":approve",
		body(`{"reviewed_by":"director","review_note":"approved"}`),
	)
	router.ServeHTTP(approveResp, approveReq)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("expected approve 200, got %d: %s", approveResp.Code, approveResp.Body.String())
	}
	var approvedPayload struct {
		ApprovalGate approvalGateResponse `json:"approval_gate"`
	}
	decodeBody(t, approveResp, &approvedPayload)
	if approvedPayload.ApprovalGate.Status != "approved" {
		t.Fatalf("expected approved gate, got %q", approvedPayload.ApprovalGate.Status)
	}
	changesResp := httptest.NewRecorder()
	changesReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/approval-gates/"+gatesPayload.ApprovalGates[1].ID+":request-changes",
		body(`{"review_note":"revise continuity"}`),
	)
	router.ServeHTTP(changesResp, changesReq)
	if changesResp.Code != http.StatusOK {
		t.Fatalf("expected request changes 200, got %d: %s", changesResp.Code, changesResp.Body.String())
	}

	exportResp := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/exports", nil)
	router.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusAccepted {
		t.Fatalf("expected export 202, got %d: %s", exportResp.Code, exportResp.Body.String())
	}
	var exportPayload struct {
		Export exportResponse `json:"export"`
	}
	decodeBody(t, exportResp, &exportPayload)
	if _, err := productionService.ProcessQueuedExports(exportReq.Context(), jobs.DefaultExecutionLimit); err != nil {
		t.Fatalf("process export job: %v", err)
	}
	exportDetailResp := httptest.NewRecorder()
	exportDetailReq := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+exportPayload.Export.ID, nil)
	router.ServeHTTP(exportDetailResp, exportDetailReq)
	if exportDetailResp.Code != http.StatusOK {
		t.Fatalf("expected export detail 200, got %d: %s", exportDetailResp.Code, exportDetailResp.Body.String())
	}
	var exportDetailPayload struct {
		Export exportResponse `json:"export"`
	}
	decodeBody(t, exportDetailResp, &exportDetailPayload)
	if exportDetailPayload.Export.Status != "succeeded" {
		t.Fatalf("expected succeeded export, got %q", exportDetailPayload.Export.Status)
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
