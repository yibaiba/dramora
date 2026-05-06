package repo

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
)

func TestSQLiteProductionRepositoryCreateStoryAnalysisRun(t *testing.T) {
	t.Parallel()

	sqliteDB, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "production.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	ctx := context.Background()
	identityRepo := NewSQLiteIdentityRepository(sqliteDB.DB)
	if err := identityRepo.CreateOrganization(ctx, CreateOrganizationParams{
		OrganizationID: "org-1",
		Name:           "Demo Workspace",
	}); err != nil {
		t.Fatalf("create organization: %v", err)
	}

	projectRepo := NewSQLiteProjectRepository(sqliteDB.DB)
	if _, err := projectRepo.CreateProject(ctx, CreateProjectParams{
		ID:             "project-1",
		OrganizationID: "org-1",
		Name:           "联调项目",
		Description:    "sqlite production repo",
		Status:         domain.ProjectStatusDraft,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := projectRepo.CreateEpisode(ctx, CreateEpisodeParams{
		ID:        "episode-1",
		ProjectID: "project-1",
		Number:    1,
		Title:     "第一集",
		Status:    domain.EpisodeStatusDraft,
	}); err != nil {
		t.Fatalf("create episode: %v", err)
	}

	productionRepo := NewSQLiteProductionRepository(sqliteDB.DB)
	run, err := productionRepo.CreateStoryAnalysisRun(ctx, CreateStoryAnalysisRunParams{
		WorkflowRunID:   "workflow-1",
		GenerationJobID: "job-1",
		ProjectID:       "project-1",
		EpisodeID:       "episode-1",
		RequestKey:      "story-analysis:episode-1",
		Provider:        "openai",
		Model:           "gpt-4.1",
		Prompt:          "analyze story",
	})
	if err != nil {
		t.Fatalf("create story analysis run: %v", err)
	}

	if run.WorkflowRun.CreatedAt.IsZero() || run.WorkflowRun.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed workflow timestamps, got created=%v updated=%v", run.WorkflowRun.CreatedAt, run.WorkflowRun.UpdatedAt)
	}
	if run.GenerationJob.CreatedAt.IsZero() || run.GenerationJob.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed generation job timestamps, got created=%v updated=%v", run.GenerationJob.CreatedAt, run.GenerationJob.UpdatedAt)
	}
	if run.GenerationJob.Status != domain.GenerationJobStatusQueued {
		t.Fatalf("expected queued generation job, got %s", run.GenerationJob.Status)
	}

	workflowRun, err := productionRepo.GetWorkflowRun(ctx, "workflow-1")
	if err != nil {
		t.Fatalf("get workflow run: %v", err)
	}
	if workflowRun.ID != "workflow-1" {
		t.Fatalf("unexpected workflow run: %+v", workflowRun)
	}

	jobs, err := productionRepo.ListGenerationJobs(ctx)
	if err != nil {
		t.Fatalf("list generation jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-1" {
		t.Fatalf("unexpected jobs: %+v", jobs)
	}
	if jobs[0].Priority != 0 || jobs[0].RetryCount != 0 || jobs[0].ParentJobID != nil {
		t.Fatalf("expected sqlite default queue fields, got %+v", jobs[0])
	}

	analysis, err := productionRepo.CreateStoryAnalysis(ctx, CreateStoryAnalysisParams{
		ID:              "analysis-1",
		ProjectID:       "project-1",
		EpisodeID:       "episode-1",
		WorkflowRunID:   "workflow-1",
		GenerationJobID: "job-1",
		Status:          domain.StoryAnalysisStatusGenerated,
		Summary:         "故事摘要",
		Themes:          []string{"成长"},
		CharacterSeeds:  []string{"主角"},
		SceneSeeds:      []string{"校园"},
		PropSeeds:       []string{"书包"},
		Outline: []domain.StoryBeat{{
			Code:       "beat-1",
			Title:      "开场",
			Summary:    "主角登场",
			VisualGoal: "建立角色",
		}},
		AgentOutputs: []domain.StoryAgentOutput{{
			Role:       "director",
			Status:     "done",
			Output:     "可拍",
			Highlights: []string{"人物关系明确"},
		}},
	})
	if err != nil {
		t.Fatalf("create story analysis: %v", err)
	}
	if analysis.CreatedAt.IsZero() || analysis.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed story analysis timestamps, got created=%v updated=%v", analysis.CreatedAt, analysis.UpdatedAt)
	}

	analyses, err := productionRepo.ListStoryAnalyses(ctx, "episode-1")
	if err != nil {
		t.Fatalf("list story analyses: %v", err)
	}
	if len(analyses) != 1 || analyses[0].ID != "analysis-1" {
		t.Fatalf("unexpected story analyses: %+v", analyses)
	}
}
