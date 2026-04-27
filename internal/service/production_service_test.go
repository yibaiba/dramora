package service

import (
	"context"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

func TestProductionServiceProcessesQueuedGenerationJobsNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo, testOrganizationID)
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

func TestProductionServiceProcessesQueuedExportsNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo, testOrganizationID)
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

	ctx := context.Background()
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

	ctx := context.Background()
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

	ctx := context.Background()
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

	ctx := context.Background()
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

	ctx := context.Background()
	projectRepo := repo.NewMemoryProjectRepository()
	projectService := NewProjectService(projectRepo, testOrganizationID)
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
