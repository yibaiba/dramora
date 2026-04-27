package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/repo"
)

type ProductionService struct {
	production repo.ProductionRepository
	jobClient  jobs.Client
}

type StartStoryAnalysisResult struct {
	WorkflowRun   domain.WorkflowRun
	GenerationJob domain.GenerationJob
}

type SaveTimelineInput struct {
	EpisodeID  string
	DurationMS int
	Tracks     []SaveTimelineTrackInput
}

type SaveTimelineTrackInput struct {
	Kind     string
	Name     string
	Position int
	Clips    []SaveTimelineClipInput
}

type SaveTimelineClipInput struct {
	AssetID     string
	Kind        string
	StartMS     int
	DurationMS  int
	TrimStartMS int
}

var noopGenerationSteps = []struct {
	status  domain.GenerationJobStatus
	message string
}{
	{domain.GenerationJobStatusSubmitting, "no-op worker submitting generation job"},
	{domain.GenerationJobStatusSubmitted, "no-op worker submitted generation job"},
	{domain.GenerationJobStatusDownloading, "no-op worker downloading generated output"},
	{domain.GenerationJobStatusPostprocessing, "no-op worker postprocessing generated output"},
}

func NewProductionService(production repo.ProductionRepository, jobClient jobs.Client) *ProductionService {
	if jobClient == nil {
		jobClient = jobs.NewNoopClient()
	}
	return &ProductionService{production: production, jobClient: jobClient}
}

func (s *ProductionService) StartStoryAnalysis(
	ctx context.Context,
	episode domain.Episode,
) (StartStoryAnalysisResult, error) {
	workflowRunID, err := domain.NewID()
	if err != nil {
		return StartStoryAnalysisResult{}, err
	}
	generationJobID, err := domain.NewID()
	if err != nil {
		return StartStoryAnalysisResult{}, err
	}

	run, err := s.production.CreateStoryAnalysisRun(ctx, repo.CreateStoryAnalysisRunParams{
		WorkflowRunID:   workflowRunID,
		GenerationJobID: generationJobID,
		ProjectID:       episode.ProjectID,
		EpisodeID:       episode.ID,
		RequestKey:      "story-analysis:" + episode.ID,
		Provider:        "internal",
		Model:           "story-analyst-agent",
		Prompt:          "Analyze episode story source and extract characters, scenes, props, and beats.",
	})
	if err != nil {
		return StartStoryAnalysisResult{}, err
	}

	if err := s.jobClient.Enqueue(ctx, jobs.Job{
		ID:   generationJobID,
		Kind: jobs.JobKindWorkflowSchedule,
		Payload: map[string]any{
			"workflow_run_id":   workflowRunID,
			"generation_job_id": generationJobID,
		},
	}); err != nil {
		return StartStoryAnalysisResult{}, err
	}

	return StartStoryAnalysisResult{
		WorkflowRun:   run.WorkflowRun,
		GenerationJob: run.GenerationJob,
	}, nil
}

func (s *ProductionService) GetWorkflowRun(ctx context.Context, id string) (domain.WorkflowRun, error) {
	if strings.TrimSpace(id) == "" {
		return domain.WorkflowRun{}, fmt.Errorf("%w: workflow run id is required", domain.ErrInvalidInput)
	}
	return s.production.GetWorkflowRun(ctx, id)
}

func (s *ProductionService) ListGenerationJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	return s.production.ListGenerationJobs(ctx)
}

func (s *ProductionService) ProcessQueuedGenerationJobs(ctx context.Context, limit int) (jobs.ExecutionSummary, error) {
	if limit <= 0 {
		return jobs.ExecutionSummary{}, fmt.Errorf("%w: execution limit must be positive", domain.ErrInvalidInput)
	}

	queuedJobs, err := s.production.ListGenerationJobsByStatus(ctx, domain.GenerationJobStatusQueued, limit)
	if err != nil {
		return jobs.ExecutionSummary{}, err
	}

	summary := jobs.ExecutionSummary{}
	for _, generationJob := range queuedJobs {
		summary.Processed++
		if err := s.processGenerationJobNoop(ctx, generationJob); err != nil {
			summary.Failed++
			return summary, fmt.Errorf("process generation job %s: %w", generationJob.ID, err)
		}
		summary.Succeeded++
	}
	return summary, nil
}

func (s *ProductionService) ProcessQueuedExports(ctx context.Context, limit int) (jobs.ExecutionSummary, error) {
	if limit <= 0 {
		return jobs.ExecutionSummary{}, fmt.Errorf("%w: execution limit must be positive", domain.ErrInvalidInput)
	}

	summary, err := s.processExportsByStatus(ctx, domain.ExportStatusQueued, limit)
	if err != nil {
		return jobs.ExecutionSummary{}, err
	}
	if remaining := limit - summary.Processed; remaining > 0 {
		renderingSummary, err := s.processExportsByStatus(ctx, domain.ExportStatusRendering, remaining)
		summary = jobs.MergeExecutionSummaries(summary, renderingSummary)
		if err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func (s *ProductionService) processExportsByStatus(
	ctx context.Context,
	status domain.ExportStatus,
	limit int,
) (jobs.ExecutionSummary, error) {
	exports, err := s.production.ListExportsByStatus(ctx, status, limit)
	if err != nil {
		return jobs.ExecutionSummary{}, err
	}

	summary := jobs.ExecutionSummary{}
	for _, export := range exports {
		summary.Processed++
		if err := s.processExportNoop(ctx, export); err != nil {
			summary.Failed++
			return summary, fmt.Errorf("process export %s: %w", export.ID, err)
		}
		summary.Succeeded++
	}
	return summary, nil
}

func (s *ProductionService) processExportNoop(ctx context.Context, export domain.Export) error {
	current := export
	if current.Status == domain.ExportStatusQueued {
		rendering, err := s.advanceExport(ctx, current, domain.ExportStatusRendering)
		if err != nil {
			return err
		}
		current = rendering
	}
	_, err := s.advanceExport(ctx, current, domain.ExportStatusSucceeded)
	return err
}

func (s *ProductionService) advanceExport(
	ctx context.Context,
	export domain.Export,
	nextStatus domain.ExportStatus,
) (domain.Export, error) {
	if err := export.Status.ValidateTransition(nextStatus); err != nil {
		return domain.Export{}, err
	}
	return s.production.AdvanceExportStatus(ctx, repo.AdvanceExportStatusParams{
		ID: export.ID, From: export.Status, To: nextStatus,
	})
}

func (s *ProductionService) processGenerationJobNoop(ctx context.Context, generationJob domain.GenerationJob) error {
	current := generationJob
	for _, step := range noopGenerationSteps {
		if err := current.Status.ValidateTransition(step.status); err != nil {
			return err
		}
		next, err := s.production.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
			ID:           current.ID,
			From:         current.Status,
			To:           step.status,
			EventMessage: step.message,
		})
		if err != nil {
			return err
		}
		current = next
	}
	if current.TaskType == "story_analysis" {
		_, err := s.completeGeneratedStoryAnalysis(ctx, current)
		return err
	}
	if err := current.Status.ValidateTransition(domain.GenerationJobStatusSucceeded); err != nil {
		return err
	}
	_, err := s.production.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
		ID:           current.ID,
		From:         current.Status,
		To:           domain.GenerationJobStatusSucceeded,
		EventMessage: "no-op worker completed generation job",
	})
	return err
}

func (s *ProductionService) GetGenerationJob(ctx context.Context, id string) (domain.GenerationJob, error) {
	if strings.TrimSpace(id) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id is required", domain.ErrInvalidInput)
	}
	return s.production.GetGenerationJob(ctx, id)
}
