package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/media"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

type ProductionService struct {
	production   repo.ProductionRepository
	jobClient    jobs.Client
	seedance     seedanceProvider
	agentSvc     *AgentService
	projectSvc   *ProjectService
	providerSvc  *ProviderService
	walletSvc    *WalletService
	mediaStorage media.Storage
	metrics      workerMetrics
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

func NewProductionService(production repo.ProductionRepository, jobClient jobs.Client) *ProductionService {
	if jobClient == nil {
		jobClient = jobs.NewNoopClient()
	}
	return &ProductionService{
		production: production,
		jobClient:  jobClient,
		seedance:   provider.NewSeedanceAdapterFromEnv(),
	}
}

func NewProductionServiceWithSeedance(
	production repo.ProductionRepository,
	jobClient jobs.Client,
	seedance seedanceProvider,
) *ProductionService {
	service := NewProductionService(production, jobClient)
	if seedance != nil {
		service.seedance = seedance
	}
	return service
}

func (s *ProductionService) SetAgentService(agentSvc *AgentService) {
	s.agentSvc = agentSvc
}

func (s *ProductionService) SetProjectService(projectSvc *ProjectService) {
	s.projectSvc = projectSvc
}

func (s *ProductionService) SetWalletService(walletSvc *WalletService) {
	s.walletSvc = walletSvc
}

// SetMediaStorage 注入媒体二进制存储后端，audio worker 在收到非 URL（仅字节）
// 结果时会把 bytes 落到该存储并使用返回的 URI 作为资产引用。未注入时回退
// 到老的 inline placeholder（manmu://providers/audio/inline?bytes=...）。
func (s *ProductionService) SetMediaStorage(storage media.Storage) {
	s.mediaStorage = storage
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
		RequestKey:      "story-analysis:" + episode.ID + ":" + generationJobID,
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
	run, err := s.production.GetWorkflowRun(ctx, id)
	if err != nil {
		return domain.WorkflowRun{}, err
	}
	if err := s.authorizeScopedResource(ctx, run.ProjectID, run.EpisodeID); err != nil {
		return domain.WorkflowRun{}, err
	}
	return run, nil
}

func (s *ProductionService) ListGenerationJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	jobs, err := s.production.ListGenerationJobs(ctx)
	if err != nil {
		return nil, err
	}
	return s.filterGenerationJobsForContext(ctx, jobs)
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
		jobCtx, ctxErr := s.workerJobAuthContextForTimeline(ctx, export.TimelineID)
		if ctxErr != nil {
			s.metrics.recordExportSkip(ctxErr.Error())
			slog.Default().Warn("worker skipped export: cannot resolve organization context",
				"export_id", export.ID,
				"timeline_id", export.TimelineID,
				"error", ctxErr,
			)
			continue
		}
		summary.Processed++
		if err := s.processExportNoop(jobCtx, export); err != nil {
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

func (s *ProductionService) GetGenerationJob(ctx context.Context, id string) (domain.GenerationJob, error) {
	if strings.TrimSpace(id) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id is required", domain.ErrInvalidInput)
	}
	job, err := s.production.GetGenerationJob(ctx, id)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if err := s.authorizeScopedResource(ctx, job.ProjectID, job.EpisodeID); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

type GenerationJobRecovery struct {
	Job     domain.GenerationJob
	Events  []domain.GenerationJobEvent
	Summary GenerationJobRecoverySummary
}

type GenerationJobRecoverySummary struct {
	IsTerminal       bool
	IsRecoverable    bool
	CurrentStatus    domain.GenerationJobStatus
	StatusEnteredAt  time.Time
	LastEventAt      time.Time
	StatusEventCount int
	TotalEventCount  int
	NextHint         string
}

func (s *ProductionService) GetGenerationJobRecovery(
	ctx context.Context,
	id string,
) (GenerationJobRecovery, error) {
	job, err := s.GetGenerationJob(ctx, id)
	if err != nil {
		return GenerationJobRecovery{}, err
	}
	events, err := s.production.ListGenerationJobEvents(ctx, job.ID, 0)
	if err != nil {
		return GenerationJobRecovery{}, err
	}
	return GenerationJobRecovery{
		Job:     job,
		Events:  events,
		Summary: buildGenerationJobRecoverySummary(job, events),
	}, nil
}

func buildGenerationJobRecoverySummary(
	job domain.GenerationJob,
	events []domain.GenerationJobEvent,
) GenerationJobRecoverySummary {
	summary := GenerationJobRecoverySummary{
		CurrentStatus:   job.Status,
		IsTerminal:      isTerminalGenerationStatus(job.Status),
		IsRecoverable:   isRecoverableGenerationStatus(job.Status),
		TotalEventCount: len(events),
	}
	for _, ev := range events {
		if ev.CreatedAt.After(summary.LastEventAt) {
			summary.LastEventAt = ev.CreatedAt
		}
		if ev.Status == job.Status {
			summary.StatusEventCount++
			if ev.CreatedAt.After(summary.StatusEnteredAt) {
				summary.StatusEnteredAt = ev.CreatedAt
			}
		}
	}
	if summary.StatusEnteredAt.IsZero() {
		summary.StatusEnteredAt = job.UpdatedAt
	}
	if summary.LastEventAt.IsZero() {
		summary.LastEventAt = job.UpdatedAt
	}
	summary.NextHint = recoveryNextHint(job, summary)
	return summary
}

func isTerminalGenerationStatus(status domain.GenerationJobStatus) bool {
	switch status {
	case domain.GenerationJobStatusSucceeded, domain.GenerationJobStatusFailed:
		return true
	}
	return false
}

func isRecoverableGenerationStatus(status domain.GenerationJobStatus) bool {
	switch status {
	case domain.GenerationJobStatusQueued,
		domain.GenerationJobStatusSubmitting,
		domain.GenerationJobStatusSubmitted,
		domain.GenerationJobStatusPolling,
		domain.GenerationJobStatusDownloading,
		domain.GenerationJobStatusPostprocessing:
		return true
	}
	return false
}

func recoveryNextHint(job domain.GenerationJob, summary GenerationJobRecoverySummary) string {
	switch job.Status {
	case domain.GenerationJobStatusSucceeded:
		return "job complete; result asset persisted"
	case domain.GenerationJobStatusFailed:
		return "job failed; create a new request to retry"
	case domain.GenerationJobStatusQueued:
		return "waiting for worker to pick up"
	case domain.GenerationJobStatusSubmitting:
		return "worker will resubmit or recover provider task on next poll"
	case domain.GenerationJobStatusSubmitted, domain.GenerationJobStatusPolling:
		return "worker will poll provider for completion on next tick"
	case domain.GenerationJobStatusDownloading, domain.GenerationJobStatusPostprocessing:
		return "worker will resume download / postprocessing on next tick"
	}
	return ""
}

type ExportRecoveryEvent struct {
	Status    domain.ExportStatus
	Message   string
	CreatedAt time.Time
}

type ExportRecoverySummary struct {
	IsTerminal      bool
	IsRecoverable   bool
	CurrentStatus   domain.ExportStatus
	StatusEnteredAt time.Time
	LastEventAt     time.Time
	TotalEventCount int
	NextHint        string
}

type ExportRecovery struct {
	Export  domain.Export
	Events  []ExportRecoveryEvent
	Summary ExportRecoverySummary
}

func (s *ProductionService) GetExportRecovery(ctx context.Context, id string) (ExportRecovery, error) {
	export, err := s.GetExport(ctx, id)
	if err != nil {
		return ExportRecovery{}, err
	}
	events := buildExportRecoveryEvents(export)
	return ExportRecovery{
		Export:  export,
		Events:  events,
		Summary: buildExportRecoverySummary(export, events),
	}, nil
}

func buildExportRecoveryEvents(export domain.Export) []ExportRecoveryEvent {
	events := []ExportRecoveryEvent{
		{Status: domain.ExportStatusQueued, Message: "export queued", CreatedAt: export.CreatedAt},
	}
	if export.Status != domain.ExportStatusQueued && !export.UpdatedAt.Equal(export.CreatedAt) {
		events = append(events, ExportRecoveryEvent{
			Status:    export.Status,
			Message:   "current status",
			CreatedAt: export.UpdatedAt,
		})
	}
	return events
}

func buildExportRecoverySummary(export domain.Export, events []ExportRecoveryEvent) ExportRecoverySummary {
	summary := ExportRecoverySummary{
		CurrentStatus:   export.Status,
		IsTerminal:      isTerminalExportStatus(export.Status),
		IsRecoverable:   isRecoverableExportStatus(export.Status),
		TotalEventCount: len(events),
		StatusEnteredAt: export.UpdatedAt,
		LastEventAt:     export.UpdatedAt,
	}
	if summary.StatusEnteredAt.IsZero() {
		summary.StatusEnteredAt = export.CreatedAt
	}
	if summary.LastEventAt.IsZero() {
		summary.LastEventAt = export.CreatedAt
	}
	for _, ev := range events {
		if ev.CreatedAt.After(summary.LastEventAt) {
			summary.LastEventAt = ev.CreatedAt
		}
	}
	summary.NextHint = exportRecoveryNextHint(export.Status)
	return summary
}

func isTerminalExportStatus(status domain.ExportStatus) bool {
	switch status {
	case domain.ExportStatusSucceeded, domain.ExportStatusFailed, domain.ExportStatusCanceled:
		return true
	}
	return false
}

func isRecoverableExportStatus(status domain.ExportStatus) bool {
	switch status {
	case domain.ExportStatusQueued, domain.ExportStatusRendering:
		return true
	}
	return false
}

func exportRecoveryNextHint(status domain.ExportStatus) string {
	switch status {
	case domain.ExportStatusQueued:
		return "waiting for worker to start rendering"
	case domain.ExportStatusRendering:
		return "worker will finalize rendering on next tick"
	case domain.ExportStatusSucceeded:
		return "export complete; artifact available"
	case domain.ExportStatusFailed:
		return "export failed; create a new request to retry"
	case domain.ExportStatusCanceled:
		return "export canceled"
	}
	return ""
}
