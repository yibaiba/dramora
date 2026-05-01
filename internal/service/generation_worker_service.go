package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

type seedanceProvider interface {
	SubmitGeneration(context.Context, provider.SeedanceRequestInput) (provider.SeedanceGenerationTask, error)
	PollGeneration(context.Context, string) (provider.SeedanceGenerationTask, error)
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

func (s *ProductionService) ProcessQueuedGenerationJobs(ctx context.Context, limit int) (jobs.ExecutionSummary, error) {
	if limit <= 0 {
		return jobs.ExecutionSummary{}, fmt.Errorf("%w: execution limit must be positive", domain.ErrInvalidInput)
	}
	generationJobs, err := s.listProcessableGenerationJobs(ctx, limit)
	if err != nil {
		return jobs.ExecutionSummary{}, err
	}

	summary := jobs.ExecutionSummary{}
	for _, generationJob := range generationJobs {
		if !shouldProcessGenerationJob(generationJob) {
			continue
		}
		jobCtx, ctxErr := s.workerJobAuthContextForProject(ctx, generationJob.ProjectID)
		if ctxErr != nil {
			s.metrics.recordGenerationSkip(ctxErr.Error())
			slog.Default().Warn("worker skipped generation job: cannot resolve organization context",
				"job_id", generationJob.ID,
				"project_id", generationJob.ProjectID,
				"error", ctxErr,
			)
			continue
		}
		summary.Processed++
		if err := s.processGenerationJob(jobCtx, generationJob); err != nil {
			summary.Failed++
			return summary, fmt.Errorf("process generation job %s: %w", generationJob.ID, err)
		}
		summary.Succeeded++
	}
	return summary, nil
}

func shouldProcessGenerationJob(generationJob domain.GenerationJob) bool {
	if isStoryAnalysisJob(generationJob) {
		switch generationJob.Status {
		case domain.GenerationJobStatusQueued,
			domain.GenerationJobStatusSubmitting,
			domain.GenerationJobStatusSubmitted,
			domain.GenerationJobStatusDownloading,
			domain.GenerationJobStatusPostprocessing:
			return true
		default:
			return false
		}
	}
	return generationJob.Status == domain.GenerationJobStatusQueued || isSeedanceVideoJob(generationJob)
}

func (s *ProductionService) listProcessableGenerationJobs(
	ctx context.Context,
	limit int,
) ([]domain.GenerationJob, error) {
	statuses := []domain.GenerationJobStatus{
		domain.GenerationJobStatusQueued,
		domain.GenerationJobStatusSubmitting,
		domain.GenerationJobStatusSubmitted,
		domain.GenerationJobStatusPolling,
		domain.GenerationJobStatusDownloading,
		domain.GenerationJobStatusPostprocessing,
	}
	collected := make([]domain.GenerationJob, 0, limit)
	for _, status := range statuses {
		remaining := limit - len(collected)
		if remaining <= 0 {
			break
		}
		generationJobs, err := s.production.ListGenerationJobsByStatus(ctx, status, remaining)
		if err != nil {
			return nil, err
		}
		collected = append(collected, generationJobs...)
	}
	return collected, nil
}

func (s *ProductionService) processGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	if isStoryAnalysisJob(generationJob) {
		return s.processStoryAnalysisJob(ctx, generationJob)
	}
	if !isSeedanceVideoJob(generationJob) {
		if generationJob.Status != domain.GenerationJobStatusQueued {
			return nil
		}
		return s.processGenerationJobNoop(ctx, generationJob)
	}
	switch generationJob.Status {
	case domain.GenerationJobStatusQueued:
		return s.submitSeedanceGenerationJob(ctx, generationJob)
	case domain.GenerationJobStatusSubmitting:
		return s.recoverSubmittingSeedanceJob(ctx, generationJob)
	case domain.GenerationJobStatusSubmitted, domain.GenerationJobStatusPolling:
		return s.pollSeedanceGenerationJob(ctx, generationJob)
	case domain.GenerationJobStatusDownloading, domain.GenerationJobStatusPostprocessing:
		return s.resumeSeedanceGenerationJob(ctx, generationJob)
	default:
		return nil
	}
}

func (s *ProductionService) processStoryAnalysisJob(ctx context.Context, generationJob domain.GenerationJob) error {
	current := generationJob
	if current.Status == domain.GenerationJobStatusQueued {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSubmitting, "", "no-op worker submitting generation job")
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusSubmitting {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSubmitted, "", "no-op worker submitted generation job")
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusSubmitted {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusDownloading, "", "no-op worker downloading generated output")
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusDownloading {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusPostprocessing, "", "no-op worker postprocessing generated output")
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusPostprocessing {
		_, err := s.completeGeneratedStoryAnalysis(ctx, current)
		return err
	}
	return nil
}

func (s *ProductionService) recoverSubmittingSeedanceJob(ctx context.Context, generationJob domain.GenerationJob) error {
	if strings.TrimSpace(generationJob.ProviderTaskID) != "" {
		_, err := s.advanceGenerationJob(
			ctx,
			generationJob,
			domain.GenerationJobStatusSubmitted,
			generationJob.ProviderTaskID,
			"seedance worker recovered submitted provider task",
		)
		return err
	}
	_, err := s.advanceGenerationJob(
		ctx,
		generationJob,
		domain.GenerationJobStatusFailed,
		"",
		"seedance worker could not recover submitting job without provider task id",
	)
	return err
}

func (s *ProductionService) submitSeedanceGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	submitting, err := s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusSubmitting, "", "seedance worker submitting generation job")
	if err != nil {
		return err
	}
	task, err := s.resolveSeedance(ctx).SubmitGeneration(ctx, seedanceRequestInput(generationJob))
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, submitting, domain.GenerationJobStatusFailed, "", "seedance worker submit failed")
		return err
	}
	_, err = s.advanceGenerationJob(
		ctx,
		submitting,
		domain.GenerationJobStatusSubmitted,
		task.ID,
		"seedance worker submitted provider task",
	)
	return err
}

func (s *ProductionService) pollSeedanceGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	task, err := s.pollSeedanceTask(ctx, generationJob)
	if err != nil {
		return err
	}
	switch seedanceTaskState(task.Status) {
	case "succeeded":
		if strings.TrimSpace(task.ResultURI) == "" {
			return fmt.Errorf("%w: seedance result uri is required", domain.ErrInvalidInput)
		}
		return s.completeSeedanceGenerationJob(ctx, generationJob, task.ResultURI)
	case "failed":
		_, err := s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusFailed, "", "seedance provider task failed")
		return err
	default:
		if generationJob.Status == domain.GenerationJobStatusSubmitted {
			_, err := s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusPolling, "", "seedance provider task still running")
			return err
		}
		return nil
	}
}

func (s *ProductionService) resumeSeedanceGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	if generationJob.Status != domain.GenerationJobStatusDownloading || generationJob.ResultAssetID != "" {
		return s.completeSeedanceGenerationJob(ctx, generationJob, "")
	}
	task, err := s.pollSeedanceTask(ctx, generationJob)
	if err != nil {
		return err
	}
	if seedanceTaskState(task.Status) != "succeeded" {
		return fmt.Errorf("%w: seedance downloading job is not complete", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(task.ResultURI) == "" {
		return fmt.Errorf("%w: seedance result uri is required", domain.ErrInvalidInput)
	}
	return s.completeSeedanceGenerationJob(ctx, generationJob, task.ResultURI)
}

func (s *ProductionService) pollSeedanceTask(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (provider.SeedanceGenerationTask, error) {
	if strings.TrimSpace(generationJob.ProviderTaskID) == "" {
		return provider.SeedanceGenerationTask{}, fmt.Errorf("%w: seedance provider task id is required", domain.ErrInvalidInput)
	}
	return s.resolveSeedance(ctx).PollGeneration(ctx, generationJob.ProviderTaskID)
}

func (s *ProductionService) completeSeedanceGenerationJob(
	ctx context.Context,
	generationJob domain.GenerationJob,
	resultURI string,
) error {
	current := generationJob
	if current.Status == domain.GenerationJobStatusSubmitted || current.Status == domain.GenerationJobStatusPolling {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusDownloading, "", "seedance worker collecting generated output")
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusDownloading {
		next, err := s.completeGenerationDownload(ctx, current, resultURI)
		if err != nil {
			return err
		}
		current = next
	}
	if current.Status == domain.GenerationJobStatusPostprocessing {
		_, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSucceeded, "", "seedance worker completed generation job")
		return err
	}
	return nil
}

func (s *ProductionService) completeGenerationDownload(
	ctx context.Context,
	generationJob domain.GenerationJob,
	resultURI string,
) (domain.GenerationJob, error) {
	if generationJob.ResultAssetID != "" {
		return s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusPostprocessing, "", "seedance worker postprocessing generated output")
	}
	if strings.TrimSpace(resultURI) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: seedance result uri is required", domain.ErrInvalidInput)
	}
	assetID, err := domain.NewID()
	if err != nil {
		return domain.GenerationJob{}, err
	}
	job, _, err := s.production.CompleteGenerationJobWithResult(ctx, repo.CompleteGenerationJobWithResultParams{
		Job: repo.AdvanceGenerationJobStatusParams{
			ID: generationJob.ID, From: generationJob.Status, To: domain.GenerationJobStatusPostprocessing,
			EventMessage: "seedance worker downloaded result asset",
		},
		Asset: repo.CreateAssetParams{
			ID: assetID, ProjectID: generationJob.ProjectID, EpisodeID: generationJob.EpisodeID,
			Kind: "video", Purpose: "generated_video", URI: generationResultAssetURI(generationJob, resultURI),
			Status: domain.AssetStatusReady,
		},
	})
	return job, err
}

func generationResultAssetURI(generationJob domain.GenerationJob, resultURI string) string {
	return strings.TrimSpace(resultURI)
}

func (s *ProductionService) processGenerationJobNoop(ctx context.Context, generationJob domain.GenerationJob) error {
	current := generationJob
	for _, step := range noopGenerationSteps {
		next, err := s.advanceGenerationJob(ctx, current, step.status, "", step.message)
		if err != nil {
			return err
		}
		current = next
	}
	if current.TaskType == "story_analysis" {
		_, err := s.completeGeneratedStoryAnalysis(ctx, current)
		return err
	}
	_, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSucceeded, "", "no-op worker completed generation job")
	return err
}

func (s *ProductionService) advanceGenerationJob(
	ctx context.Context,
	generationJob domain.GenerationJob,
	nextStatus domain.GenerationJobStatus,
	providerTaskID string,
	message string,
) (domain.GenerationJob, error) {
	if err := generationJob.Status.ValidateTransition(nextStatus); err != nil {
		return domain.GenerationJob{}, err
	}
	return s.production.AdvanceGenerationJobStatus(ctx, repo.AdvanceGenerationJobStatusParams{
		ID: generationJob.ID, From: generationJob.Status, To: nextStatus,
		ProviderTaskID: providerTaskID, EventMessage: message,
	})
}

func isSeedanceVideoJob(generationJob domain.GenerationJob) bool {
	if generationJob.Provider != provider.ProviderSeedance {
		return false
	}
	switch provider.TaskType(generationJob.TaskType) {
	case provider.TaskTypeTextToVideo, provider.TaskTypeImageToVideo, provider.TaskTypeFirstLast:
		return true
	default:
		return false
	}
}

func isStoryAnalysisJob(generationJob domain.GenerationJob) bool {
	return generationJob.TaskType == "story_analysis"
}

func seedanceRequestInput(generationJob domain.GenerationJob) provider.SeedanceRequestInput {
	return provider.SeedanceRequestInput{
		Prompt:      generationJob.Prompt,
		TaskType:    provider.TaskType(generationJob.TaskType),
		Ratio:       stringParam(generationJob.Params, "ratio"),
		Resolution:  stringParam(generationJob.Params, "resolution"),
		DurationSec: intParam(generationJob.Params, "duration"),
		Seed:        intParam(generationJob.Params, "seed"),
		References:  seedanceReferenceTokens(generationJob.Params["reference_bindings"]),
	}
}

func seedanceTaskState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "completed", "complete", "finished", "done":
		return "succeeded"
	case "failed", "failure", "error", "canceled", "cancelled":
		return "failed"
	default:
		return "running"
	}
}

func seedanceReferenceTokens(value any) []provider.SeedanceRefToken {
	switch refs := value.(type) {
	case []domain.PromptReferenceBinding:
		tokens := make([]provider.SeedanceRefToken, 0, len(refs))
		for _, ref := range refs {
			tokens = appendSeedanceToken(tokens, ref.Token, ref.Role, ref.URI)
		}
		return tokens
	case []any:
		tokens := make([]provider.SeedanceRefToken, 0, len(refs))
		for _, ref := range refs {
			if refMap, ok := ref.(map[string]any); ok {
				tokens = appendSeedanceToken(tokens, stringMapParam(refMap, "Token", "token"), stringMapParam(refMap, "Role", "role"), stringMapParam(refMap, "URI", "uri"))
			}
		}
		return tokens
	default:
		return nil
	}
}

func appendSeedanceToken(tokens []provider.SeedanceRefToken, token string, role string, uri string) []provider.SeedanceRefToken {
	if strings.TrimSpace(uri) == "" {
		return tokens
	}
	return append(tokens, provider.SeedanceRefToken{Token: token, Role: role, URL: uri})
}

func stringParam(params map[string]any, key string) string {
	value, _ := params[key].(string)
	return value
}

func stringMapParam(params map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := params[key].(string); ok {
			return value
		}
	}
	return ""
}

func intParam(params map[string]any, key string) int {
	switch value := params[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
