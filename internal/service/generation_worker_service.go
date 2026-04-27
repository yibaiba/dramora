package service

import (
	"context"
	"fmt"
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
		summary.Processed++
		if err := s.processGenerationJob(ctx, generationJob); err != nil {
			summary.Failed++
			return summary, fmt.Errorf("process generation job %s: %w", generationJob.ID, err)
		}
		summary.Succeeded++
	}
	return summary, nil
}

func shouldProcessGenerationJob(generationJob domain.GenerationJob) bool {
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
		return s.completeSeedanceGenerationJob(ctx, generationJob)
	default:
		return nil
	}
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
	task, err := s.seedance.SubmitGeneration(ctx, seedanceRequestInput(generationJob))
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
	if strings.TrimSpace(generationJob.ProviderTaskID) == "" {
		return fmt.Errorf("%w: seedance provider task id is required", domain.ErrInvalidInput)
	}
	task, err := s.seedance.PollGeneration(ctx, generationJob.ProviderTaskID)
	if err != nil {
		return err
	}
	switch seedanceTaskState(task.Status) {
	case "succeeded":
		return s.completeSeedanceGenerationJob(ctx, generationJob)
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

func (s *ProductionService) completeSeedanceGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	current := generationJob
	steps := []struct {
		status  domain.GenerationJobStatus
		message string
	}{
		{domain.GenerationJobStatusDownloading, "seedance worker collecting generated output"},
		{domain.GenerationJobStatusPostprocessing, "seedance worker postprocessing generated output"},
		{domain.GenerationJobStatusSucceeded, "seedance worker completed generation job"},
	}
	for _, step := range remainingGenerationSteps(current.Status, steps) {
		next, err := s.advanceGenerationJob(ctx, current, step.status, "", step.message)
		if err != nil {
			return err
		}
		current = next
	}
	return nil
}

func remainingGenerationSteps(
	currentStatus domain.GenerationJobStatus,
	steps []struct {
		status  domain.GenerationJobStatus
		message string
	},
) []struct {
	status  domain.GenerationJobStatus
	message string
} {
	for index, step := range steps {
		if currentStatus == step.status {
			return steps[index+1:]
		}
	}
	return steps
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
