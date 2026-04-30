package service

import (
	"context"
	"sync"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/workflow"
)

func TestProductionServiceProcessesQueuedGenerationJobsNoop(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo)
	productionService := NewProductionService(repo.NewMemoryProductionRepository(), nil)

	project, err := projectService.CreateProject(ctx, CreateProjectInput{Name: "Worker Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(ctx, CreateEpisodeInput{
		ProjectID: project.ID,
		Title:     "Worker Episode",
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := productionService.StartStoryAnalysis(ctx, episode); err != nil {
		t.Fatalf("start story analysis: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process queued generation jobs: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	generationJobs, err := productionService.ListGenerationJobs(ctx)
	if err != nil {
		t.Fatalf("list generation jobs: %v", err)
	}
	if got := generationJobs[0].Status; got != domain.GenerationJobStatusSucceeded {
		t.Fatalf("expected succeeded job, got %q", got)
	}
	workflowRun, err := productionService.GetWorkflowRun(ctx, generationJobs[0].WorkflowRunID)
	if err != nil {
		t.Fatalf("get workflow run: %v", err)
	}
	if workflowRun.Status != domain.WorkflowRunStatusSucceeded {
		t.Fatalf("expected workflow run succeeded, got %+v", workflowRun)
	}
	detail, err := productionService.GetWorkflowRunDetail(ctx, generationJobs[0].WorkflowRunID)
	if err != nil {
		t.Fatalf("get workflow run detail: %v", err)
	}
	if detail.Checkpoint == nil || detail.Checkpoint.CompletedNodes != len(workflow.Phase1Graph.Nodes) {
		t.Fatalf("expected local story analysis checkpoint summary, got %+v", detail.Checkpoint)
	}
	if len(detail.NodeRuns) != len(workflow.Phase1Graph.Nodes) {
		t.Fatalf("expected local story analysis node runs, got %+v", detail.NodeRuns)
	}

	analyses, err := productionService.ListStoryAnalyses(ctx, episode.ID)
	if err != nil {
		t.Fatalf("list story analyses: %v", err)
	}
	if len(analyses) != 1 {
		t.Fatalf("expected 1 story analysis, got %d", len(analyses))
	}
	if len(analyses[0].CharacterSeeds) == 0 || len(analyses[0].SceneSeeds) == 0 {
		t.Fatalf("expected story analysis seeds, got %+v", analyses[0])
	}
}

func TestProductionServiceResumesStoryAnalysisFromCheckpoint(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo)
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)

	var (
		mu    sync.Mutex
		calls []string
	)
	productionService.SetAgentService(&AgentService{
		availabilityFunc: func(context.Context) bool { return true },
		executorFactory: func(_ string) workflow.NodeExecutor {
			return func(_ context.Context, nodeID string, _ workflow.NodeKind, bb *workflow.Blackboard) (any, error) {
				mu.Lock()
				calls = append(calls, nodeID)
				mu.Unlock()
				if nodeID == "outline_planner" {
					value, ok := bb.Read("story_analyst")
					if !ok {
						t.Fatalf("expected restored story_analyst output")
					}
					result, ok := value.(*AgentResult)
					if !ok || result.Output != "restored-story-output" {
						t.Fatalf("expected restored AgentResult, got %#v", value)
					}
				}
				result := &AgentResult{
					Role:       nodeID,
					Output:     nodeID + "-output",
					Highlights: []string{nodeID + "-highlight"},
				}
				bb.Write(nodeID, result)
				return result, nil
			}
		},
	})

	project, err := projectService.CreateProject(ctx, CreateProjectInput{Name: "Checkpoint Resume"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(ctx, CreateEpisodeInput{ProjectID: project.ID, Title: "Resume Episode"})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := productionRepo.CreateStorySource(ctx, repo.CreateStorySourceParams{
		ID:          "00000000-0000-0000-0000-00000000ca11",
		ProjectID:   project.ID,
		EpisodeID:   episode.ID,
		SourceType:  "novel",
		Title:       "恢复测试",
		ContentText: "主角在废墟中找到了遗失的徽章。",
		Language:    "zh-CN",
	}); err != nil {
		t.Fatalf("create story source: %v", err)
	}

	started, err := productionService.StartStoryAnalysis(ctx, episode)
	if err != nil {
		t.Fatalf("start story analysis: %v", err)
	}
	current := started.GenerationJob
	for _, status := range []domain.GenerationJobStatus{
		domain.GenerationJobStatusSubmitting,
		domain.GenerationJobStatusSubmitted,
		domain.GenerationJobStatusDownloading,
		domain.GenerationJobStatusPostprocessing,
	} {
		current, err = productionRepo.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
			ID: current.ID, From: current.Status, To: status, EventMessage: "resume setup",
		})
		if err != nil {
			t.Fatalf("advance generation job to %s: %v", status, err)
		}
	}

	checkpointStore := newStoryAnalysisCheckpointStore(productionRepo)
	if err := checkpointStore.Save(ctx, started.WorkflowRun.ID, &workflow.Checkpoint{
		WorkflowID: started.WorkflowRun.ID,
		Sequence:   7,
		Runs: map[string]workflow.NodeRunSnapshot{
			"story_analyst": {
				NodeID: "story_analyst",
				Kind:   workflow.NodeKindStoryAnalysis,
				Status: workflow.NodeSucceeded,
				Output: &AgentResult{
					Role:       "story_analyst",
					Output:     "restored-story-output",
					Highlights: []string{"theme"},
				},
			},
		},
		Blackboard: map[string]any{
			"story_analyst": &AgentResult{
				Role:       "story_analyst",
				Output:     "restored-story-output",
				Highlights: []string{"theme"},
			},
		},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("resume story analysis job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	mu.Lock()
	recordedCalls := append([]string(nil), calls...)
	mu.Unlock()
	for _, unexpected := range []string{"story_analyst"} {
		for _, call := range recordedCalls {
			if call == unexpected {
				t.Fatalf("expected checkpoint resume to skip %s, calls=%v", unexpected, recordedCalls)
			}
		}
	}
	for _, expected := range []string{"outline_planner", "character_analyst", "scene_analyst", "prop_analyst"} {
		found := false
		for _, call := range recordedCalls {
			if call == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected resumed execution to include %s, calls=%v", expected, recordedCalls)
		}
	}

	workflowRun, err := productionService.GetWorkflowRun(ctx, started.WorkflowRun.ID)
	if err != nil {
		t.Fatalf("get workflow run: %v", err)
	}
	if workflowRun.Status != domain.WorkflowRunStatusSucceeded {
		t.Fatalf("expected workflow run succeeded, got %+v", workflowRun)
	}
	detail, err := productionService.GetWorkflowRunDetail(ctx, started.WorkflowRun.ID)
	if err != nil {
		t.Fatalf("get workflow run detail: %v", err)
	}
	if detail.Checkpoint == nil {
		t.Fatal("expected checkpoint summary")
	}
	if detail.Checkpoint.CompletedNodes != len(workflow.Phase1Graph.Nodes) || detail.Checkpoint.FailedNodes != 0 {
		t.Fatalf("unexpected checkpoint summary: %+v", detail.Checkpoint)
	}
	if len(detail.Checkpoint.BlackboardRoles) == 0 {
		t.Fatalf("expected checkpoint summary blackboard roles, got %+v", detail.Checkpoint)
	}
	if len(detail.NodeRuns) != len(workflow.Phase1Graph.Nodes) {
		t.Fatalf("expected node details for each workflow node, got %+v", detail.NodeRuns)
	}
	if detail.NodeRuns[0].NodeID != "story_analyst" || detail.NodeRuns[0].Status != domain.WorkflowNodeRunStatusSucceeded {
		t.Fatalf("expected ordered succeeded node details, got %+v", detail.NodeRuns[0])
	}
	if len(detail.NodeRuns[0].Highlights) == 0 {
		t.Fatalf("expected node highlights in recovery detail, got %+v", detail.NodeRuns[0])
	}
	analyses, err := productionService.ListStoryAnalyses(ctx, episode.ID)
	if err != nil {
		t.Fatalf("list analyses: %v", err)
	}
	if len(analyses) != 1 {
		t.Fatalf("expected 1 analysis after resume, got %d", len(analyses))
	}
	if len(analyses[0].AgentOutputs) != len(workflow.Phase1Graph.Nodes) {
		t.Fatalf("expected full agent outputs after resume, got %+v", analyses[0].AgentOutputs)
	}
}

func TestProductionServiceProcessesQueuedExportsNoop(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo)
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)

	project, err := projectService.CreateProject(ctx, CreateProjectInput{Name: "Export Worker Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(ctx, CreateEpisodeInput{ProjectID: project.ID, Title: "Export Episode"})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := productionService.SaveEpisodeTimeline(ctx, SaveTimelineInput{EpisodeID: episode.ID, DurationMS: 3000}); err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	createdExport, err := productionService.StartEpisodeExport(ctx, episode.ID)
	if err != nil {
		t.Fatalf("start export: %v", err)
	}

	summary, err := productionService.ProcessQueuedExports(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process queued exports: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	export, err := productionService.GetExport(ctx, createdExport.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if export.Status != domain.ExportStatusSucceeded {
		t.Fatalf("expected succeeded export, got %q", export.Status)
	}
}

func TestProductionServiceSubmitsAndPollsSeedanceGenerationJob(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionServiceWithSeedance(productionRepo, nil, provider.NewSeedanceAdapter("", "", nil))
	jobID := "00000000-0000-0000-0000-000000000201"

	createdJob, err := productionRepo.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: "00000000-0000-0000-0000-000000000202",
		EpisodeID:  "00000000-0000-0000-0000-000000000203",
		RequestKey: "shot-video:test", Provider: provider.ProviderSeedance,
		Model: provider.ModelSeedance10ProFast, TaskType: string(provider.TaskTypeImageToVideo),
		Status: domain.GenerationJobStatusQueued, Prompt: "SH001 opening shot",
		Params: map[string]any{
			"ratio": "16:9", "resolution": "720p", "duration": 5,
			"reference_bindings": []domain.PromptReferenceBinding{
				{Token: "@image1", Role: "first_frame", URI: "manmu://asset/one"},
			},
		},
		EventMessage: "shot video generation queued",
	})
	if err != nil {
		t.Fatalf("create generation job: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("submit queued generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected submit summary: %+v", summary)
	}
	submittedJob, err := productionService.GetGenerationJob(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("get submitted job: %v", err)
	}
	if submittedJob.Status != domain.GenerationJobStatusSubmitted || submittedJob.ProviderTaskID == "" {
		t.Fatalf("expected submitted job with provider task id, got %+v", submittedJob)
	}

	summary, err = productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("poll submitted generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected poll summary: %+v", summary)
	}
	completedJob, err := productionService.GetGenerationJob(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("get completed job: %v", err)
	}
	if completedJob.Status != domain.GenerationJobStatusSucceeded {
		t.Fatalf("expected succeeded job, got %+v", completedJob)
	}
	if completedJob.ResultAssetID == "" {
		t.Fatalf("expected completed job to persist a result asset id, got %+v", completedJob)
	}
	assets, err := productionService.ListEpisodeAssets(ctx, completedJob.EpisodeID)
	if err != nil {
		t.Fatalf("list result assets: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 result asset, got %+v", assets)
	}
	if assets[0].ID != completedJob.ResultAssetID || assets[0].Kind != "video" || assets[0].Status != domain.AssetStatusReady {
		t.Fatalf("expected ready video result asset linked to job, job=%+v asset=%+v", completedJob, assets[0])
	}
}

func TestProductionServiceRecoversInterruptedSeedanceGenerationJob(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionServiceWithSeedance(productionRepo, nil, provider.NewSeedanceAdapter("", "", nil))
	jobID := "00000000-0000-0000-0000-000000000211"

	createdJob, err := productionRepo.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: "00000000-0000-0000-0000-000000000212",
		EpisodeID:  "00000000-0000-0000-0000-000000000213",
		RequestKey: "shot-video:interrupted", Provider: provider.ProviderSeedance,
		Model: provider.ModelSeedance10ProFast, TaskType: string(provider.TaskTypeTextToVideo),
		Status: domain.GenerationJobStatusQueued, Prompt: "SH002 interrupted shot",
		Params:       map[string]any{"duration": 5},
		EventMessage: "shot video generation queued",
	})
	if err != nil {
		t.Fatalf("create generation job: %v", err)
	}
	submittingJob, err := productionRepo.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
		ID: createdJob.ID, From: domain.GenerationJobStatusQueued, To: domain.GenerationJobStatusSubmitting,
		EventMessage: "seedance worker submitting generation job",
	})
	if err != nil {
		t.Fatalf("advance job to submitting: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("recover submitting generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected recovery summary: %+v", summary)
	}
	recoveredJob, err := productionService.GetGenerationJob(ctx, submittingJob.ID)
	if err != nil {
		t.Fatalf("get recovered job: %v", err)
	}
	if recoveredJob.Status != domain.GenerationJobStatusFailed {
		t.Fatalf("expected unrecoverable submitting job to fail, got %+v", recoveredJob)
	}
}

func TestProductionServiceResumesDownloadingSeedanceJobWithResultAsset(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionServiceWithSeedance(productionRepo, nil, fakeSeedanceProvider{
		pollTask: provider.SeedanceGenerationTask{
			ID: "provider-task-1", Status: "succeeded", ResultURI: "https://cdn.example.test/shot-1.mp4",
		},
	})
	job := createSeedanceVideoJob(t, ctx, productionRepo, "00000000-0000-0000-0000-000000000221")
	job = advanceGenerationJobForTest(t, ctx, productionRepo, job, domain.GenerationJobStatusSubmitting, "")
	job = advanceGenerationJobForTest(t, ctx, productionRepo, job, domain.GenerationJobStatusSubmitted, "provider-task-1")
	job = advanceGenerationJobForTest(t, ctx, productionRepo, job, domain.GenerationJobStatusDownloading, "")

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("resume downloading generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	completedJob, err := productionService.GetGenerationJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("get completed job: %v", err)
	}
	if completedJob.Status != domain.GenerationJobStatusSucceeded || completedJob.ResultAssetID == "" {
		t.Fatalf("expected succeeded job with result asset, got %+v", completedJob)
	}
	assets, err := productionService.ListEpisodeAssets(ctx, completedJob.EpisodeID)
	if err != nil {
		t.Fatalf("list result assets: %v", err)
	}
	if len(assets) != 1 || assets[0].URI != "https://cdn.example.test/shot-1.mp4" {
		t.Fatalf("expected recovered provider result asset, got %+v", assets)
	}
}

func TestMemoryProductionRepositoryDoesNotCreateResultAssetWhenJobTransitionFails(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	job := createSeedanceVideoJob(t, ctx, productionRepo, "00000000-0000-0000-0000-000000000231")

	_, _, err := productionRepo.CompleteGenerationJobWithResult(ctx, repo.CompleteGenerationJobWithResultParams{
		Job: repo.AdvanceGenerationJobStatusParams{
			ID: job.ID, From: domain.GenerationJobStatusDownloading, To: domain.GenerationJobStatusPostprocessing,
			EventMessage: "seedance worker downloaded result asset",
		},
		Asset: repo.CreateAssetParams{
			ID: "00000000-0000-0000-0000-000000000232", ProjectID: job.ProjectID, EpisodeID: job.EpisodeID,
			Kind: "video", Purpose: "generated_video", URI: "https://cdn.example.test/orphan.mp4",
			Status: domain.AssetStatusReady,
		},
	})
	if err == nil {
		t.Fatal("expected transition mismatch error")
	}
	assets, err := productionRepo.ListAssetsByEpisode(ctx, job.EpisodeID)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 0 {
		t.Fatalf("expected no orphan assets after failed transition, got %+v", assets)
	}
}

func TestProductionServiceResumesRenderingExportsNoop(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo)
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)

	project, err := projectService.CreateProject(ctx, CreateProjectInput{Name: "Resume Export Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	episode, err := projectService.CreateEpisode(ctx, CreateEpisodeInput{ProjectID: project.ID, Title: "Resume Export"})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := productionService.SaveEpisodeTimeline(ctx, SaveTimelineInput{EpisodeID: episode.ID, DurationMS: 3000}); err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	createdExport, err := productionService.StartEpisodeExport(ctx, episode.ID)
	if err != nil {
		t.Fatalf("start export: %v", err)
	}
	if _, err := productionRepo.AdvanceExportStatus(ctx, repo.AdvanceExportStatusParams{
		ID: createdExport.ID, From: domain.ExportStatusQueued, To: domain.ExportStatusRendering,
	}); err != nil {
		t.Fatalf("advance export to rendering: %v", err)
	}

	summary, err := productionService.ProcessQueuedExports(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process rendering export: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	export, err := productionService.GetExport(ctx, createdExport.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if export.Status != domain.ExportStatusSucceeded {
		t.Fatalf("expected resumed export to succeed, got %q", export.Status)
	}
}

type fakeSeedanceProvider struct {
	pollTask provider.SeedanceGenerationTask
}

func (p fakeSeedanceProvider) SubmitGeneration(
	context.Context,
	provider.SeedanceRequestInput,
) (provider.SeedanceGenerationTask, error) {
	return provider.SeedanceGenerationTask{ID: "provider-task-1", Status: "queued", Mode: "fake"}, nil
}

func (p fakeSeedanceProvider) PollGeneration(context.Context, string) (provider.SeedanceGenerationTask, error) {
	return p.pollTask, nil
}

func createSeedanceVideoJob(
	t *testing.T,
	ctx context.Context,
	productionRepo *repo.MemoryProductionRepository,
	jobID string,
) domain.GenerationJob {
	t.Helper()
	job, err := productionRepo.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: "00000000-0000-0000-0000-000000000301",
		EpisodeID:  "00000000-0000-0000-0000-000000000302",
		RequestKey: "shot-video:" + jobID, Provider: provider.ProviderSeedance,
		Model: provider.ModelSeedance10ProFast, TaskType: string(provider.TaskTypeTextToVideo),
		Status: domain.GenerationJobStatusQueued, Prompt: "Seedance shot",
		Params:       map[string]any{"duration": 5},
		EventMessage: "shot video generation queued",
	})
	if err != nil {
		t.Fatalf("create generation job: %v", err)
	}
	return job
}

func advanceGenerationJobForTest(
	t *testing.T,
	ctx context.Context,
	productionRepo *repo.MemoryProductionRepository,
	job domain.GenerationJob,
	next domain.GenerationJobStatus,
	providerTaskID string,
) domain.GenerationJob {
	t.Helper()
	advanced, err := productionRepo.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
		ID: job.ID, From: job.Status, To: next, ProviderTaskID: providerTaskID,
		EventMessage: "test generation job advance",
	})
	if err != nil {
		t.Fatalf("advance generation job to %s: %v", next, err)
	}
	return advanced
}
