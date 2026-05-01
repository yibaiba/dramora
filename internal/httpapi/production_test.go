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

	recoveryResp := httptest.NewRecorder()
	recoveryReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/generation-jobs/"+payload.GenerationJob.ID+"/recovery",
		nil,
	)
	router.ServeHTTP(recoveryResp, recoveryReq)
	if recoveryResp.Code != http.StatusOK {
		t.Fatalf("expected recovery 200, got %d: %s", recoveryResp.Code, recoveryResp.Body.String())
	}
	var recoveryPayload struct {
		Recovery generationJobRecoveryResponse `json:"generation_job_recovery"`
	}
	decodeBody(t, recoveryResp, &recoveryPayload)
	if recoveryPayload.Recovery.Job.ID != payload.GenerationJob.ID {
		t.Fatalf("expected recovery job id %q, got %q", payload.GenerationJob.ID, recoveryPayload.Recovery.Job.ID)
	}
	if recoveryPayload.Recovery.Summary.TotalEventCount == 0 {
		t.Fatalf("expected at least one lifecycle event, got %+v", recoveryPayload.Recovery.Summary)
	}
	if !recoveryPayload.Recovery.Summary.IsRecoverable && !recoveryPayload.Recovery.Summary.IsTerminal {
		t.Fatalf("expected recoverable or terminal summary, got %+v", recoveryPayload.Recovery.Summary)
	}
	if recoveryPayload.Recovery.Summary.NextHint == "" {
		t.Fatalf("expected next_hint, got empty")
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

	sourceResp := httptest.NewRecorder()
	sourceReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/episodes/"+episode.ID+"/story-sources",
		body(`{"source_type":"novel","title":"天门试炼","content_text":"云澜在天门前发现玉佩线索。白璃提醒他宗门长老已经隐瞒真相。黑龙压境时，云澜必须守护同伴并完成试炼。","language":"zh-CN"}`),
	)
	router.ServeHTTP(sourceResp, sourceReq)
	if sourceResp.Code != http.StatusCreated {
		t.Fatalf("expected source 201, got %d: %s", sourceResp.Code, sourceResp.Body.String())
	}
	var sourcePayload struct {
		StorySource storySourceResponse `json:"story_source"`
	}
	decodeBody(t, sourceResp, &sourcePayload)
	if sourcePayload.StorySource.Title != "天门试炼" {
		t.Fatalf("expected saved source title, got %q", sourcePayload.StorySource.Title)
	}
	sourceListResp := httptest.NewRecorder()
	sourceListReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/story-sources", nil)
	router.ServeHTTP(sourceListResp, sourceListReq)
	if sourceListResp.Code != http.StatusOK {
		t.Fatalf("expected source list 200, got %d: %s", sourceListResp.Code, sourceListResp.Body.String())
	}

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
	if payload.StoryAnalyses[0].StorySourceID != sourcePayload.StorySource.ID {
		t.Fatalf("expected analysis linked to source %q, got %q", sourcePayload.StorySource.ID, payload.StoryAnalyses[0].StorySourceID)
	}
	if len(payload.StoryAnalyses[0].Outline) == 0 || len(payload.StoryAnalyses[0].AgentOutputs) < 5 {
		t.Fatalf("expected outline and multi-agent outputs, got %+v", payload.StoryAnalyses[0])
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

	workflowResp := httptest.NewRecorder()
	workflowReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workflow-runs/"+payload.StoryAnalyses[0].WorkflowRunID,
		nil,
	)
	router.ServeHTTP(workflowResp, workflowReq)
	if workflowResp.Code != http.StatusOK {
		t.Fatalf("expected workflow detail 200, got %d: %s", workflowResp.Code, workflowResp.Body.String())
	}
	var workflowPayload struct {
		WorkflowRun workflowRunResponse `json:"workflow_run"`
	}
	decodeBody(t, workflowResp, &workflowPayload)
	if workflowPayload.WorkflowRun.ID != payload.StoryAnalyses[0].WorkflowRunID {
		t.Fatalf("expected workflow run %q, got %+v", payload.StoryAnalyses[0].WorkflowRunID, workflowPayload.WorkflowRun)
	}
	if workflowPayload.WorkflowRun.CheckpointSummary == nil {
		t.Fatalf("expected workflow checkpoint summary, got %+v", workflowPayload.WorkflowRun)
	}
	if len(workflowPayload.WorkflowRun.NodeRuns) < 5 {
		t.Fatalf("expected workflow node runs, got %+v", workflowPayload.WorkflowRun.NodeRuns)
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
	saveBibleResp := httptest.NewRecorder()
	saveBibleReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/story-map-characters/"+storyMapPayload.StoryMap.Characters[0].ID+"/character-bible:save",
		body(`{"character_bible":{"anchor":"Maya，黑色长发配蓝色挑染，绿色眼睛，左眉有细疤。","palette":{"skin":"#E8C9A0","hair":"#1A1A2E","accent":"#3B82F6","eyes":"#22C55E","costume":"#1F2937"},"expressions":["中性","惊讶"],"reference_angles":["正面","3/4 左"],"reference_assets":[],"wardrobe":"C01: 黑色战斗服","notes":"保持蓝色挑染和护腕细节"}}`),
	)
	router.ServeHTTP(saveBibleResp, saveBibleReq)
	if saveBibleResp.Code != http.StatusOK {
		t.Fatalf("expected character bible 200, got %d: %s", saveBibleResp.Code, saveBibleResp.Body.String())
	}
	var saveBiblePayload struct {
		StoryMapItem characterResponse `json:"story_map_item"`
	}
	decodeBody(t, saveBibleResp, &saveBiblePayload)
	if saveBiblePayload.StoryMapItem.CharacterBible == nil || saveBiblePayload.StoryMapItem.CharacterBible.Anchor == "" {
		t.Fatalf("expected persisted character bible, got %+v", saveBiblePayload.StoryMapItem)
	}
	readStoryMapResp := httptest.NewRecorder()
	readStoryMapReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/story-map", nil)
	router.ServeHTTP(readStoryMapResp, readStoryMapReq)
	if readStoryMapResp.Code != http.StatusOK {
		t.Fatalf("expected story map get 200, got %d: %s", readStoryMapResp.Code, readStoryMapResp.Body.String())
	}
	var readStoryMapPayload struct {
		StoryMap storyMapResponse `json:"story_map"`
	}
	decodeBody(t, readStoryMapResp, &readStoryMapPayload)
	if readStoryMapPayload.StoryMap.Characters[0].CharacterBible == nil || readStoryMapPayload.StoryMap.Characters[0].CharacterBible.Anchor == "" {
		t.Fatalf("expected character bible on story map read, got %+v", readStoryMapPayload.StoryMap.Characters[0])
	}
	workspaceResp := httptest.NewRecorder()
	workspaceReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/storyboard-workspace", nil)
	router.ServeHTTP(workspaceResp, workspaceReq)
	if workspaceResp.Code != http.StatusOK {
		t.Fatalf("expected workspace 200, got %d: %s", workspaceResp.Code, workspaceResp.Body.String())
	}
	var workspacePayload struct {
		StoryboardWorkspace storyboardWorkspaceResponse `json:"storyboard_workspace"`
	}
	decodeBody(t, workspaceResp, &workspacePayload)
	if workspacePayload.StoryboardWorkspace.StoryMap.Characters[0].CharacterBible == nil {
		t.Fatalf("expected character bible in storyboard workspace, got %+v", workspacePayload.StoryboardWorkspace.StoryMap.Characters[0])
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
	characterAssetID := ""
	for _, asset := range assetsPayload.Assets {
		if asset.Kind == "character" && asset.Purpose == storyMapPayload.StoryMap.Characters[0].Code {
			characterAssetID = asset.ID
			break
		}
	}
	if characterAssetID == "" {
		t.Fatalf("expected locked candidate for %s, got %+v", storyMapPayload.StoryMap.Characters[0].Code, assetsPayload.Assets)
	}

	saveUnlockedReferenceResp := httptest.NewRecorder()
	saveUnlockedReferenceReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/story-map-characters/"+storyMapPayload.StoryMap.Characters[0].ID+"/character-bible:save",
		body(`{"character_bible":{"anchor":"Maya，黑色长发配蓝色挑染，绿色眼睛，左眉有细疤。","palette":{"skin":"#E8C9A0","hair":"#1A1A2E","accent":"#3B82F6","eyes":"#22C55E","costume":"#1F2937"},"expressions":["中性","惊讶"],"reference_angles":["正面","3/4 左"],"reference_assets":[{"angle":"正面","asset_id":"`+characterAssetID+`"}],"wardrobe":"C01: 黑色战斗服","notes":"保持蓝色挑染和护腕细节"}}`),
	)
	router.ServeHTTP(saveUnlockedReferenceResp, saveUnlockedReferenceReq)
	if saveUnlockedReferenceResp.Code != http.StatusBadRequest {
		t.Fatalf("expected unlocked reference asset 400, got %d: %s", saveUnlockedReferenceResp.Code, saveUnlockedReferenceResp.Body.String())
	}

	lockResp := httptest.NewRecorder()
	lockReq := httptest.NewRequest(http.MethodPost, "/api/v1/assets/"+characterAssetID+":lock", nil)
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
	saveLockedReferenceResp := httptest.NewRecorder()
	saveLockedReferenceReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/story-map-characters/"+storyMapPayload.StoryMap.Characters[0].ID+"/character-bible:save",
		body(`{"character_bible":{"anchor":"Maya，黑色长发配蓝色挑染，绿色眼睛，左眉有细疤。","palette":{"skin":"#E8C9A0","hair":"#1A1A2E","accent":"#3B82F6","eyes":"#22C55E","costume":"#1F2937"},"expressions":["中性","惊讶"],"reference_angles":["正面","3/4 左"],"reference_assets":[{"angle":"正面","asset_id":"`+characterAssetID+`"}],"wardrobe":"C01: 黑色战斗服","notes":"保持蓝色挑染和护腕细节"}}`),
	)
	router.ServeHTTP(saveLockedReferenceResp, saveLockedReferenceReq)
	if saveLockedReferenceResp.Code != http.StatusOK {
		t.Fatalf("expected locked reference asset 200, got %d: %s", saveLockedReferenceResp.Code, saveLockedReferenceResp.Body.String())
	}
	var saveLockedReferencePayload struct {
		StoryMapItem characterResponse `json:"story_map_item"`
	}
	decodeBody(t, saveLockedReferenceResp, &saveLockedReferencePayload)
	if saveLockedReferencePayload.StoryMapItem.CharacterBible == nil || len(saveLockedReferencePayload.StoryMapItem.CharacterBible.ReferenceAssets) != 1 {
		t.Fatalf("expected persisted reference asset mapping, got %+v", saveLockedReferencePayload.StoryMapItem)
	}
	readStoryMapWithReferenceResp := httptest.NewRecorder()
	readStoryMapWithReferenceReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/story-map", nil)
	router.ServeHTTP(readStoryMapWithReferenceResp, readStoryMapWithReferenceReq)
	if readStoryMapWithReferenceResp.Code != http.StatusOK {
		t.Fatalf("expected story map get 200 after reference save, got %d: %s", readStoryMapWithReferenceResp.Code, readStoryMapWithReferenceResp.Body.String())
	}
	var readStoryMapWithReferencePayload struct {
		StoryMap storyMapResponse `json:"story_map"`
	}
	decodeBody(t, readStoryMapWithReferenceResp, &readStoryMapWithReferencePayload)
	if len(readStoryMapWithReferencePayload.StoryMap.Characters[0].CharacterBible.ReferenceAssets) != 1 ||
		readStoryMapWithReferencePayload.StoryMap.Characters[0].CharacterBible.ReferenceAssets[0].AssetID != characterAssetID {
		t.Fatalf("expected reference asset on story map read, got %+v", readStoryMapWithReferencePayload.StoryMap.Characters[0].CharacterBible)
	}
	workspaceWithReferenceResp := httptest.NewRecorder()
	workspaceWithReferenceReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/storyboard-workspace", nil)
	router.ServeHTTP(workspaceWithReferenceResp, workspaceWithReferenceReq)
	if workspaceWithReferenceResp.Code != http.StatusOK {
		t.Fatalf("expected workspace 200 after reference save, got %d: %s", workspaceWithReferenceResp.Code, workspaceWithReferenceResp.Body.String())
	}
	var workspaceWithReferencePayload struct {
		StoryboardWorkspace storyboardWorkspaceResponse `json:"storyboard_workspace"`
	}
	decodeBody(t, workspaceWithReferenceResp, &workspaceWithReferencePayload)
	if len(workspaceWithReferencePayload.StoryboardWorkspace.StoryMap.Characters[0].CharacterBible.ReferenceAssets) != 1 {
		t.Fatalf("expected reference asset in storyboard workspace, got %+v", workspaceWithReferencePayload.StoryboardWorkspace.StoryMap.Characters[0].CharacterBible)
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

	packRecoveryResp := httptest.NewRecorder()
	packRecoveryReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/storyboard-shots/"+shotPayload.StoryboardShots[0].ID+"/prompt-pack/recovery",
		nil,
	)
	router.ServeHTTP(packRecoveryResp, packRecoveryReq)
	if packRecoveryResp.Code != http.StatusOK {
		t.Fatalf("expected prompt pack recovery 200, got %d: %s", packRecoveryResp.Code, packRecoveryResp.Body.String())
	}
	var packRecoveryPayload struct {
		Recovery promptPackRecoveryResponse `json:"prompt_pack_recovery"`
	}
	decodeBody(t, packRecoveryResp, &packRecoveryPayload)
	if packRecoveryPayload.Recovery.Summary.JobsTotal == 0 {
		t.Fatalf("expected prompt pack recovery to surface generation jobs, got %+v", packRecoveryPayload.Recovery.Summary)
	}
	foundJob := false
	for _, j := range packRecoveryPayload.Recovery.Jobs {
		if j.Job.ID == videoPayload.GenerationJob.ID {
			foundJob = true
			break
		}
	}
	if !foundJob {
		t.Fatalf("expected video job %q in prompt pack recovery, got %+v", videoPayload.GenerationJob.ID, packRecoveryPayload.Recovery.Jobs)
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
	var changesPayload struct {
		ApprovalGate approvalGateResponse `json:"approval_gate"`
	}
	decodeBody(t, changesResp, &changesPayload)
	if changesPayload.ApprovalGate.Status != "changes_requested" {
		t.Fatalf("expected changes requested gate, got %q", changesPayload.ApprovalGate.Status)
	}
	resubmitResp := httptest.NewRecorder()
	resubmitReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/approval-gates/"+gatesPayload.ApprovalGates[1].ID+":resubmit",
		body(`{"review_note":"updated prompts and references"}`),
	)
	router.ServeHTTP(resubmitResp, resubmitReq)
	if resubmitResp.Code != http.StatusOK {
		t.Fatalf("expected resubmit 200, got %d: %s", resubmitResp.Code, resubmitResp.Body.String())
	}
	var resubmitPayload struct {
		ApprovalGate approvalGateResponse `json:"approval_gate"`
	}
	decodeBody(t, resubmitResp, &resubmitPayload)
	if resubmitPayload.ApprovalGate.Status != "pending" {
		t.Fatalf("expected resubmitted gate to return to pending, got %q", resubmitPayload.ApprovalGate.Status)
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

	exportRecoveryResp := httptest.NewRecorder()
	exportRecoveryReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/exports/"+exportPayload.Export.ID+"/recovery",
		nil,
	)
	router.ServeHTTP(exportRecoveryResp, exportRecoveryReq)
	if exportRecoveryResp.Code != http.StatusOK {
		t.Fatalf("expected export recovery 200, got %d: %s", exportRecoveryResp.Code, exportRecoveryResp.Body.String())
	}
	var exportRecoveryPayload struct {
		Recovery exportRecoveryResponse `json:"export_recovery"`
	}
	decodeBody(t, exportRecoveryResp, &exportRecoveryPayload)
	if !exportRecoveryPayload.Recovery.Summary.IsTerminal {
		t.Fatalf("expected terminal export recovery summary, got %+v", exportRecoveryPayload.Recovery.Summary)
	}
	if exportRecoveryPayload.Recovery.Summary.NextHint == "" {
		t.Fatalf("expected non-empty next_hint")
	}
}

func TestSeedEpisodeProductionRoute(t *testing.T) {
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

	seedResp := httptest.NewRecorder()
	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/production:seed", nil)
	router.ServeHTTP(seedResp, seedReq)
	if seedResp.Code != http.StatusCreated {
		t.Fatalf("expected production seed 201, got %d: %s", seedResp.Code, seedResp.Body.String())
	}
	var payload struct {
		ApprovalGates   []approvalGateResponse   `json:"approval_gates"`
		Assets          []assetResponse          `json:"assets"`
		StoryMap        storyMapResponse         `json:"story_map"`
		StoryboardShots []storyboardShotResponse `json:"storyboard_shots"`
	}
	decodeBody(t, seedResp, &payload)
	if len(payload.StoryMap.Characters) == 0 || len(payload.StoryMap.Scenes) == 0 {
		t.Fatalf("expected production seed to create story map, got %+v", payload.StoryMap)
	}
	if len(payload.Assets) == 0 {
		t.Fatal("expected production seed to create asset candidates")
	}
	if len(payload.StoryboardShots) == 0 {
		t.Fatal("expected production seed to create storyboard shots")
	}
	if len(payload.ApprovalGates) == 0 {
		t.Fatal("expected production seed to create approval gates")
	}
}

func TestGetStoryboardWorkspaceRoute(t *testing.T) {
	t.Parallel()

	router, productionService := testRouterWithProductionService()
	episode := createTestEpisode(t, router)

	startResp := httptest.NewRecorder()
	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/story-analysis/start", nil)
	router.ServeHTTP(startResp, startReq)
	if startResp.Code != http.StatusAccepted {
		t.Fatalf("expected story analysis 202, got %d: %s", startResp.Code, startResp.Body.String())
	}
	if _, err := productionService.ProcessQueuedGenerationJobs(startReq.Context(), jobs.DefaultExecutionLimit); err != nil {
		t.Fatalf("process story analysis job: %v", err)
	}

	for _, path := range []string{
		"/api/v1/episodes/" + episode.ID + "/story-map:seed",
		"/api/v1/episodes/" + episode.ID + "/assets:seed",
		"/api/v1/episodes/" + episode.ID + "/storyboard-shots:seed",
		"/api/v1/episodes/" + episode.ID + "/approval-gates:seed",
	} {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected %s to return 201, got %d: %s", path, resp.Code, resp.Body.String())
		}
	}

	assetsResp := httptest.NewRecorder()
	assetsReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/assets", nil)
	router.ServeHTTP(assetsResp, assetsReq)
	if assetsResp.Code != http.StatusOK {
		t.Fatalf("expected assets 200, got %d: %s", assetsResp.Code, assetsResp.Body.String())
	}
	var assetsPayload struct {
		Assets []assetResponse `json:"assets"`
	}
	decodeBody(t, assetsResp, &assetsPayload)
	lockResp := httptest.NewRecorder()
	lockReq := httptest.NewRequest(http.MethodPost, "/api/v1/assets/"+assetsPayload.Assets[0].ID+":lock", nil)
	router.ServeHTTP(lockResp, lockReq)
	if lockResp.Code != http.StatusOK {
		t.Fatalf("expected lock 200, got %d: %s", lockResp.Code, lockResp.Body.String())
	}

	shotsResp := httptest.NewRecorder()
	shotsReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/storyboard-shots", nil)
	router.ServeHTTP(shotsResp, shotsReq)
	if shotsResp.Code != http.StatusOK {
		t.Fatalf("expected shots 200, got %d: %s", shotsResp.Code, shotsResp.Body.String())
	}
	var shotsPayload struct {
		StoryboardShots []storyboardShotResponse `json:"storyboard_shots"`
	}
	decodeBody(t, shotsResp, &shotsPayload)
	promptResp := httptest.NewRecorder()
	promptReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotsPayload.StoryboardShots[0].ID+"/prompt-pack:generate",
		nil,
	)
	router.ServeHTTP(promptResp, promptReq)
	if promptResp.Code != http.StatusCreated {
		t.Fatalf("expected prompt pack 201, got %d: %s", promptResp.Code, promptResp.Body.String())
	}
	videoResp := httptest.NewRecorder()
	videoReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storyboard-shots/"+shotsPayload.StoryboardShots[0].ID+"/videos:generate",
		nil,
	)
	router.ServeHTTP(videoResp, videoReq)
	if videoResp.Code != http.StatusAccepted {
		t.Fatalf("expected video generation 202, got %d: %s", videoResp.Code, videoResp.Body.String())
	}

	workspaceResp := httptest.NewRecorder()
	workspaceReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/storyboard-workspace", nil)
	router.ServeHTTP(workspaceResp, workspaceReq)
	if workspaceResp.Code != http.StatusOK {
		t.Fatalf("expected workspace 200, got %d: %s", workspaceResp.Code, workspaceResp.Body.String())
	}

	var payload struct {
		StoryboardWorkspace storyboardWorkspaceResponse `json:"storyboard_workspace"`
	}
	decodeBody(t, workspaceResp, &payload)
	if payload.StoryboardWorkspace.EpisodeID != episode.ID {
		t.Fatalf("expected workspace episode %q, got %q", episode.ID, payload.StoryboardWorkspace.EpisodeID)
	}
	if payload.StoryboardWorkspace.Summary.AnalysisCount == 0 || !payload.StoryboardWorkspace.Summary.StoryMapReady {
		t.Fatalf("expected workspace summary to include analysis/story map readiness, got %+v", payload.StoryboardWorkspace.Summary)
	}
	if len(payload.StoryboardWorkspace.StoryboardShots) == 0 {
		t.Fatal("expected workspace storyboard shots")
	}
	firstShot := payload.StoryboardWorkspace.StoryboardShots[0]
	if firstShot.Scene == nil || firstShot.Scene.Code == "" {
		t.Fatalf("expected workspace shot scene metadata, got %+v", firstShot)
	}
	if firstShot.PromptPack == nil || firstShot.PromptPack.Preset != "sd2_fast" {
		t.Fatalf("expected workspace shot prompt pack summary, got %+v", firstShot.PromptPack)
	}
	if firstShot.LatestGenerationJob == nil || firstShot.LatestGenerationJob.TaskType != "image_to_video" {
		t.Fatalf("expected workspace shot latest generation job, got %+v", firstShot.LatestGenerationJob)
	}
	if len(payload.StoryboardWorkspace.GenerationJobs) == 0 {
		t.Fatal("expected workspace generation jobs")
	}
}

func TestGetStoryboardWorkspaceRouteReturnsEmptyWorkspaceBeforeSeeding(t *testing.T) {
	t.Parallel()

	router, _ := testRouterWithProductionService()
	episode := createTestEpisode(t, router)

	workspaceResp := httptest.NewRecorder()
	workspaceReq := httptest.NewRequest(http.MethodGet, "/api/v1/episodes/"+episode.ID+"/storyboard-workspace", nil)
	router.ServeHTTP(workspaceResp, workspaceReq)
	if workspaceResp.Code != http.StatusOK {
		t.Fatalf("expected workspace 200, got %d: %s", workspaceResp.Code, workspaceResp.Body.String())
	}

	var payload struct {
		StoryboardWorkspace storyboardWorkspaceResponse `json:"storyboard_workspace"`
	}
	decodeBody(t, workspaceResp, &payload)
	if payload.StoryboardWorkspace.EpisodeID != episode.ID {
		t.Fatalf("expected workspace episode %q, got %q", episode.ID, payload.StoryboardWorkspace.EpisodeID)
	}
	if payload.StoryboardWorkspace.Summary.AnalysisCount != 0 || payload.StoryboardWorkspace.Summary.StoryMapReady {
		t.Fatalf("expected empty workspace summary, got %+v", payload.StoryboardWorkspace.Summary)
	}
	if len(payload.StoryboardWorkspace.StoryMap.Characters) != 0 ||
		len(payload.StoryboardWorkspace.StoryMap.Scenes) != 0 ||
		len(payload.StoryboardWorkspace.StoryMap.Props) != 0 {
		t.Fatalf("expected empty story map, got %+v", payload.StoryboardWorkspace.StoryMap)
	}
	if len(payload.StoryboardWorkspace.StoryboardShots) != 0 {
		t.Fatalf("expected no storyboard shots, got %d", len(payload.StoryboardWorkspace.StoryboardShots))
	}
	if len(payload.StoryboardWorkspace.Assets) != 0 {
		t.Fatalf("expected no assets, got %d", len(payload.StoryboardWorkspace.Assets))
	}
	if len(payload.StoryboardWorkspace.ApprovalGates) != 0 {
		t.Fatalf("expected no approval gates, got %d", len(payload.StoryboardWorkspace.ApprovalGates))
	}
	if len(payload.StoryboardWorkspace.GenerationJobs) != 0 {
		t.Fatalf("expected no generation jobs, got %d", len(payload.StoryboardWorkspace.GenerationJobs))
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
