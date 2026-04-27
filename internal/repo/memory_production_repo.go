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
	analyses  map[string]domain.StoryAnalysis
	timelines map[string]domain.Timeline
	chars     map[string]domain.Character
	scenes    map[string]domain.Scene
	props     map[string]domain.Prop
	shots     map[string]domain.StoryboardShot
	exports   map[string]domain.Export
}

func NewMemoryProductionRepository() *MemoryProductionRepository {
	return &MemoryProductionRepository{
		runs:      make(map[string]domain.WorkflowRun),
		jobs:      make(map[string]domain.GenerationJob),
		analyses:  make(map[string]domain.StoryAnalysis),
		timelines: make(map[string]domain.Timeline),
		chars:     make(map[string]domain.Character),
		scenes:    make(map[string]domain.Scene),
		props:     make(map[string]domain.Prop),
		shots:     make(map[string]domain.StoryboardShot),
		exports:   make(map[string]domain.Export),
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

func (r *MemoryProductionRepository) CompleteStoryAnalysisJob(
	_ context.Context,
	params CompleteStoryAnalysisJobParams,
) (StoryAnalysisCompletion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[params.Job.ID]
	if !ok || job.Status != params.Job.From {
		return StoryAnalysisCompletion{}, domain.ErrNotFound
	}
	job.Status = params.Job.To
	job.UpdatedAt = time.Now().UTC()
	r.jobs[job.ID] = job

	analysis := r.buildStoryAnalysisLocked(params.Analysis)
	r.analyses[analysis.ID] = analysis
	return StoryAnalysisCompletion{GenerationJob: job, StoryAnalysis: analysis}, nil
}

func (r *MemoryProductionRepository) CreateStoryAnalysis(
	_ context.Context,
	params CreateStoryAnalysisParams,
) (domain.StoryAnalysis, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	analysis := r.buildStoryAnalysisLocked(params)
	r.analyses[analysis.ID] = analysis
	return analysis, nil
}

func (r *MemoryProductionRepository) buildStoryAnalysisLocked(
	params CreateStoryAnalysisParams,
) domain.StoryAnalysis {
	now := time.Now().UTC()
	return domain.StoryAnalysis{
		ID:              params.ID,
		ProjectID:       params.ProjectID,
		EpisodeID:       params.EpisodeID,
		WorkflowRunID:   params.WorkflowRunID,
		GenerationJobID: params.GenerationJobID,
		Version:         r.nextStoryAnalysisVersion(params.EpisodeID),
		Status:          params.Status,
		Summary:         params.Summary,
		Themes:          append([]string{}, params.Themes...),
		CharacterSeeds:  append([]string{}, params.CharacterSeeds...),
		SceneSeeds:      append([]string{}, params.SceneSeeds...),
		PropSeeds:       append([]string{}, params.PropSeeds...),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (r *MemoryProductionRepository) ListStoryAnalyses(
	_ context.Context,
	episodeID string,
) ([]domain.StoryAnalysis, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	analyses := make([]domain.StoryAnalysis, 0)
	for _, analysis := range r.analyses {
		if analysis.EpisodeID == episodeID {
			analyses = append(analyses, analysis)
		}
	}
	sort.Slice(analyses, func(i int, j int) bool {
		return analyses[i].Version > analyses[j].Version
	})
	return analyses, nil
}

func (r *MemoryProductionRepository) GetStoryAnalysis(
	_ context.Context,
	analysisID string,
) (domain.StoryAnalysis, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	analysis, ok := r.analyses[analysisID]
	if !ok {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analysis, nil
}

func (r *MemoryProductionRepository) SaveStoryMap(
	_ context.Context,
	params SaveStoryMapParams,
) (StoryMap, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	storyMap := StoryMap{}
	for _, item := range params.Characters {
		value := domain.Character{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			StoryAnalysisID: item.StoryAnalysisID, Code: item.Code,
			Name: item.Name, Description: item.Description, CreatedAt: now, UpdatedAt: now,
		}
		r.chars[value.ID] = value
		storyMap.Characters = append(storyMap.Characters, value)
	}
	for _, item := range params.Scenes {
		value := domain.Scene{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			StoryAnalysisID: item.StoryAnalysisID, Code: item.Code,
			Name: item.Name, Description: item.Description, CreatedAt: now, UpdatedAt: now,
		}
		r.scenes[value.ID] = value
		storyMap.Scenes = append(storyMap.Scenes, value)
	}
	for _, item := range params.Props {
		value := domain.Prop{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			StoryAnalysisID: item.StoryAnalysisID, Code: item.Code,
			Name: item.Name, Description: item.Description, CreatedAt: now, UpdatedAt: now,
		}
		r.props[value.ID] = value
		storyMap.Props = append(storyMap.Props, value)
	}
	return storyMap, nil
}

func (r *MemoryProductionRepository) GetStoryMap(_ context.Context, episodeID string) (StoryMap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storyMap := StoryMap{}
	for _, value := range r.chars {
		if value.EpisodeID == episodeID {
			storyMap.Characters = append(storyMap.Characters, value)
		}
	}
	for _, value := range r.scenes {
		if value.EpisodeID == episodeID {
			storyMap.Scenes = append(storyMap.Scenes, value)
		}
	}
	for _, value := range r.props {
		if value.EpisodeID == episodeID {
			storyMap.Props = append(storyMap.Props, value)
		}
	}
	sortStoryMap(storyMap)
	return storyMap, nil
}

func sortStoryMap(storyMap StoryMap) {
	sort.Slice(storyMap.Characters, func(i int, j int) bool {
		return storyMap.Characters[i].Code < storyMap.Characters[j].Code
	})
	sort.Slice(storyMap.Scenes, func(i int, j int) bool {
		return storyMap.Scenes[i].Code < storyMap.Scenes[j].Code
	})
	sort.Slice(storyMap.Props, func(i int, j int) bool {
		return storyMap.Props[i].Code < storyMap.Props[j].Code
	})
}

func (r *MemoryProductionRepository) SaveStoryboardShots(
	_ context.Context,
	params SaveStoryboardShotsParams,
) ([]domain.StoryboardShot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	shots := make([]domain.StoryboardShot, 0, len(params.Shots))
	for _, item := range params.Shots {
		shot := domain.StoryboardShot{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			StoryAnalysisID: item.StoryAnalysisID, SceneID: item.SceneID,
			Code: item.Code, Title: item.Title, Description: item.Description,
			Prompt: item.Prompt, Position: item.Position, DurationMS: item.DurationMS,
			CreatedAt: now, UpdatedAt: now,
		}
		r.shots[shot.ID] = shot
		shots = append(shots, shot)
	}
	return shots, nil
}

func (r *MemoryProductionRepository) ListStoryboardShots(
	_ context.Context,
	episodeID string,
) ([]domain.StoryboardShot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	shots := make([]domain.StoryboardShot, 0)
	for _, shot := range r.shots {
		if shot.EpisodeID == episodeID {
			shots = append(shots, shot)
		}
	}
	sort.Slice(shots, func(i int, j int) bool {
		return shots[i].Position < shots[j].Position
	})
	return shots, nil
}

func (r *MemoryProductionRepository) nextStoryAnalysisVersion(episodeID string) int {
	next := 1
	for _, analysis := range r.analyses {
		if analysis.EpisodeID == episodeID && analysis.Version >= next {
			next = analysis.Version + 1
		}
	}
	return next
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

func (r *MemoryProductionRepository) SaveEpisodeTimelineGraph(
	_ context.Context,
	params SaveEpisodeTimelineGraphParams,
) (domain.Timeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	timeline, ok := r.timelines[params.EpisodeID]
	if !ok {
		timeline = domain.Timeline{ID: params.ID, EpisodeID: params.EpisodeID, Version: 1, CreatedAt: now}
	} else {
		timeline.Version++
	}
	timeline.Status = params.Status
	timeline.DurationMS = params.DurationMS
	timeline.UpdatedAt = now
	timeline.Tracks = buildMemoryTimelineTracks(params.ID, params.Tracks, now)
	r.timelines[params.EpisodeID] = timeline
	return timeline, nil
}

func buildMemoryTimelineTracks(
	timelineID string,
	params []SaveTimelineTrackParams,
	now time.Time,
) []domain.TimelineTrack {
	tracks := make([]domain.TimelineTrack, 0, len(params))
	for _, item := range params {
		track := domain.TimelineTrack{
			ID: item.ID, TimelineID: timelineID, Kind: item.Kind, Name: item.Name,
			Position: item.Position, CreatedAt: now, UpdatedAt: now,
		}
		for _, clip := range item.Clips {
			track.Clips = append(track.Clips, domain.TimelineClip{
				ID: clip.ID, TimelineID: timelineID, TrackID: item.ID,
				AssetID: clip.AssetID, Kind: clip.Kind, StartMS: clip.StartMS,
				DurationMS: clip.DurationMS, TrimStartMS: clip.TrimStartMS,
				CreatedAt: now, UpdatedAt: now,
			})
		}
		tracks = append(tracks, track)
	}
	return tracks
}

func (r *MemoryProductionRepository) CreateExport(
	_ context.Context,
	params CreateExportParams,
) (domain.Export, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	export := domain.Export{
		ID: params.ID, TimelineID: params.TimelineID, Status: params.Status,
		Format: params.Format, CreatedAt: now, UpdatedAt: now,
	}
	r.exports[export.ID] = export
	return export, nil
}

func (r *MemoryProductionRepository) GetExport(_ context.Context, exportID string) (domain.Export, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	export, ok := r.exports[exportID]
	if !ok {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, nil
}
