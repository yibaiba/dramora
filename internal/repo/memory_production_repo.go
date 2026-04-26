package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryProductionRepository struct {
	mu        sync.RWMutex
	runs      map[string]domain.WorkflowRun
	jobs      map[string]domain.GenerationJob
	timelines map[string]domain.Timeline
}

func NewMemoryProductionRepository() *MemoryProductionRepository {
	return &MemoryProductionRepository{
		runs:      make(map[string]domain.WorkflowRun),
		jobs:      make(map[string]domain.GenerationJob),
		timelines: make(map[string]domain.Timeline),
	}
}

func (r *MemoryProductionRepository) CreateStoryAnalysisRun(
	_ context.Context,
	params CreateStoryAnalysisRunParams,
) (StoryAnalysisRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	run := domain.WorkflowRun{
		ID:        params.WorkflowRunID,
		ProjectID: params.ProjectID,
		EpisodeID: params.EpisodeID,
		Status:    domain.WorkflowRunStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
	job := domain.GenerationJob{
		ID:            params.GenerationJobID,
		ProjectID:     params.ProjectID,
		EpisodeID:     params.EpisodeID,
		WorkflowRunID: params.WorkflowRunID,
		Provider:      params.Provider,
		Model:         params.Model,
		TaskType:      "story_analysis",
		Status:        domain.GenerationJobStatusQueued,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	r.runs[run.ID] = run
	r.jobs[job.ID] = job
	return StoryAnalysisRun{WorkflowRun: run, GenerationJob: job}, nil
}

func (r *MemoryProductionRepository) GetWorkflowRun(
	_ context.Context,
	workflowRunID string,
) (domain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.runs[workflowRunID]
	if !ok {
		return domain.WorkflowRun{}, domain.ErrNotFound
	}
	return run, nil
}

func (r *MemoryProductionRepository) ListGenerationJobs(_ context.Context) ([]domain.GenerationJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]domain.GenerationJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *MemoryProductionRepository) ListGenerationJobsByStatus(
	_ context.Context,
	status domain.GenerationJobStatus,
	limit int,
) ([]domain.GenerationJob, error) {
	if limit <= 0 {
		return []domain.GenerationJob{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]domain.GenerationJob, 0)
	for _, job := range r.jobs {
		if job.Status != status {
			continue
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i int, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

func (r *MemoryProductionRepository) GetGenerationJob(
	_ context.Context,
	generationJobID string,
) (domain.GenerationJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[generationJobID]
	if !ok {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	return job, nil
}

func (r *MemoryProductionRepository) AdvanceGenerationJobStatus(
	_ context.Context,
	params AdvanceGenerationJobStatusParams,
) (domain.GenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[params.ID]
	if !ok || job.Status != params.From {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	job.Status = params.To
	job.UpdatedAt = time.Now().UTC()
	r.jobs[job.ID] = job
	return job, nil
}

func (r *MemoryProductionRepository) GetEpisodeTimeline(
	_ context.Context,
	episodeID string,
) (domain.Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	timeline, ok := r.timelines[episodeID]
	if !ok {
		return domain.Timeline{}, domain.ErrNotFound
	}
	return timeline, nil
}

func (r *MemoryProductionRepository) SaveEpisodeTimeline(
	_ context.Context,
	params SaveEpisodeTimelineParams,
) (domain.Timeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	timeline, ok := r.timelines[params.EpisodeID]
	if ok {
		timeline.Status = params.Status
		timeline.DurationMS = params.DurationMS
		timeline.Version++
		timeline.UpdatedAt = now
		r.timelines[params.EpisodeID] = timeline
		return timeline, nil
	}

	timeline = domain.Timeline{
		ID:         params.ID,
		EpisodeID:  params.EpisodeID,
		Status:     params.Status,
		Version:    1,
		DurationMS: params.DurationMS,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	r.timelines[params.EpisodeID] = timeline
	return timeline, nil
}
