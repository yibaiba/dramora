package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryProductionRepository struct {
	mu             sync.RWMutex
	runs           map[string]domain.WorkflowRun
	runCheckpoints map[string][]byte
	jobs           map[string]domain.GenerationJob
	jobKeys        map[string]string
	jobEvents      map[string][]domain.GenerationJobEvent
	sources        map[string]domain.StorySource
	gates          map[string]domain.ApprovalGate
	analyses       map[string]domain.StoryAnalysis
	timelines      map[string]domain.Timeline
	chars          map[string]domain.Character
	scenes         map[string]domain.Scene
	props          map[string]domain.Prop
	shots          map[string]domain.StoryboardShot
	prompts        map[string]domain.ShotPromptPack
	assets         map[string]domain.Asset
	exports        map[string]domain.Export
}

func NewMemoryProductionRepository() *MemoryProductionRepository {
	return &MemoryProductionRepository{
		runs:           make(map[string]domain.WorkflowRun),
		runCheckpoints: make(map[string][]byte),
		jobs:           make(map[string]domain.GenerationJob),
		jobKeys:        make(map[string]string),
		jobEvents:      make(map[string][]domain.GenerationJobEvent),
		sources:        make(map[string]domain.StorySource),
		gates:          make(map[string]domain.ApprovalGate),
		analyses:       make(map[string]domain.StoryAnalysis),
		timelines:      make(map[string]domain.Timeline),
		chars:          make(map[string]domain.Character),
		scenes:         make(map[string]domain.Scene),
		props:          make(map[string]domain.Prop),
		shots:          make(map[string]domain.StoryboardShot),
		prompts:        make(map[string]domain.ShotPromptPack),
		assets:         make(map[string]domain.Asset),
		exports:        make(map[string]domain.Export),
	}
}

func (r *MemoryProductionRepository) CreateStorySource(
	_ context.Context,
	params CreateStorySourceParams,
) (domain.StorySource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	source := domain.StorySource{
		ID: params.ID, ProjectID: params.ProjectID, EpisodeID: params.EpisodeID,
		SourceType: params.SourceType, Title: params.Title, ContentText: params.ContentText,
		Language: params.Language, CreatedAt: now, UpdatedAt: now,
	}
	r.sources[source.ID] = source
	return source, nil
}

func (r *MemoryProductionRepository) ListStorySources(
	_ context.Context,
	episodeID string,
) ([]domain.StorySource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources := r.storySourcesLocked(episodeID)
	return sources, nil
}

func (r *MemoryProductionRepository) LatestStorySource(
	_ context.Context,
	episodeID string,
) (domain.StorySource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources := r.storySourcesLocked(episodeID)
	if len(sources) == 0 {
		return domain.StorySource{}, domain.ErrNotFound
	}
	return sources[0], nil
}

func (r *MemoryProductionRepository) storySourcesLocked(episodeID string) []domain.StorySource {
	sources := make([]domain.StorySource, 0)
	for _, source := range r.sources {
		if source.EpisodeID == episodeID {
			sources = append(sources, source)
		}
	}
	sort.Slice(sources, func(i int, j int) bool {
		return sources[i].CreatedAt.After(sources[j].CreatedAt)
	})
	return sources
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
		Prompt:        params.Prompt,
		Params:        map[string]any{},
		ResultAssetID: "",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	r.runs[run.ID] = run
	r.jobs[job.ID] = job
	r.jobKeys[params.RequestKey] = job.ID
	r.appendJobEventLocked(job.ID, job.Status, "story analysis queued", now)
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

func (r *MemoryProductionRepository) SaveWorkflowCheckpoint(
	_ context.Context,
	workflowRunID string,
	payload []byte,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.runs[workflowRunID]; !ok {
		return domain.ErrNotFound
	}
	r.runCheckpoints[workflowRunID] = append([]byte(nil), payload...)
	return nil
}

func (r *MemoryProductionRepository) LoadWorkflowCheckpoint(
	_ context.Context,
	workflowRunID string,
) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.runs[workflowRunID]; !ok {
		return nil, domain.ErrNotFound
	}
	payload := r.runCheckpoints[workflowRunID]
	return append([]byte(nil), payload...), nil
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

func (r *MemoryProductionRepository) CreateGenerationJob(
	_ context.Context,
	params CreateGenerationJobParams,
) (domain.GenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID := r.jobKeys[params.RequestKey]; existingID != "" {
		return r.jobs[existingID], nil
	}
	now := time.Now().UTC()
	job := domain.GenerationJob{
		ID: params.ID, ProjectID: params.ProjectID, EpisodeID: params.EpisodeID,
		WorkflowRunID: params.WorkflowRunID, Provider: params.Provider, Model: params.Model,
		TaskType: params.TaskType, Status: params.Status, Prompt: params.Prompt, Params: params.Params,
		CreatedAt: now, UpdatedAt: now,
	}
	if job.Params == nil {
		job.Params = map[string]any{}
	}
	r.jobs[job.ID] = job
	r.jobKeys[params.RequestKey] = job.ID
	r.appendJobEventLocked(job.ID, job.Status, params.EventMessage, now)
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
	if params.ProviderTaskID != "" {
		job.ProviderTaskID = params.ProviderTaskID
	}
	if params.ResultAssetID != "" {
		job.ResultAssetID = params.ResultAssetID
	}
	now := time.Now().UTC()
	job.UpdatedAt = now
	r.jobs[job.ID] = job
	r.appendJobEventLocked(job.ID, job.Status, params.EventMessage, now)
	return job, nil
}

func (r *MemoryProductionRepository) CompleteGenerationJobWithResult(
	_ context.Context,
	params CompleteGenerationJobWithResultParams,
) (domain.GenerationJob, domain.Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[params.Job.ID]
	if !ok || job.Status != params.Job.From {
		return domain.GenerationJob{}, domain.Asset{}, domain.ErrNotFound
	}
	asset := r.findOrCreateAssetLocked(params.Asset)
	job.Status = params.Job.To
	job.ResultAssetID = asset.ID
	if params.Job.ProviderTaskID != "" {
		job.ProviderTaskID = params.Job.ProviderTaskID
	}
	now := time.Now().UTC()
	job.UpdatedAt = now
	r.jobs[job.ID] = job
	r.appendJobEventLocked(job.ID, job.Status, params.Job.EventMessage, now)
	return job, asset, nil
}

func (r *MemoryProductionRepository) ListGenerationJobEvents(
	_ context.Context,
	generationJobID string,
	limit int,
) ([]domain.GenerationJobEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.jobEvents[generationJobID]
	if len(events) == 0 {
		return []domain.GenerationJobEvent{}, nil
	}
	out := make([]domain.GenerationJobEvent, len(events))
	copy(out, events)
	sort.Slice(out, func(i int, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (r *MemoryProductionRepository) appendJobEventLocked(
	jobID string,
	status domain.GenerationJobStatus,
	message string,
	createdAt time.Time,
) {
	if jobID == "" {
		return
	}
	if r.jobEvents == nil {
		r.jobEvents = make(map[string][]domain.GenerationJobEvent)
	}
	id, _ := domain.NewID()
	r.jobEvents[jobID] = append(r.jobEvents[jobID], domain.GenerationJobEvent{
		ID:              id,
		GenerationJobID: jobID,
		Status:          status,
		Message:         message,
		CreatedAt:       createdAt,
	})
}

func (r *MemoryProductionRepository) ListApprovalGates(
	_ context.Context,
	episodeID string,
) ([]domain.ApprovalGate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gates := make([]domain.ApprovalGate, 0)
	for _, gate := range r.gates {
		if gate.EpisodeID == episodeID {
			gates = append(gates, gate)
		}
	}
	sort.Slice(gates, func(i int, j int) bool {
		if gates[i].CreatedAt.Equal(gates[j].CreatedAt) {
			return gates[i].GateType < gates[j].GateType
		}
		return gates[i].CreatedAt.Before(gates[j].CreatedAt)
	})
	return gates, nil
}

func (r *MemoryProductionRepository) GetApprovalGate(
	_ context.Context,
	gateID string,
) (domain.ApprovalGate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gate, ok := r.gates[gateID]
	if !ok {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return gate, nil
}

func (r *MemoryProductionRepository) SaveApprovalGate(
	_ context.Context,
	params SaveApprovalGateParams,
) (domain.ApprovalGate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, gate := range r.gates {
		if approvalGateMatches(gate, params) {
			return gate, nil
		}
	}
	now := time.Now().UTC()
	gate := domain.ApprovalGate{
		ID: params.ID, ProjectID: params.ProjectID, EpisodeID: params.EpisodeID,
		WorkflowRunID: params.WorkflowRunID, GateType: params.GateType,
		SubjectType: params.SubjectType, SubjectID: params.SubjectID,
		Status: params.Status, CreatedAt: now, UpdatedAt: now,
	}
	r.gates[gate.ID] = gate
	return gate, nil
}

func (r *MemoryProductionRepository) ReviewApprovalGate(
	_ context.Context,
	params ReviewApprovalGateParams,
) (domain.ApprovalGate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	gate, ok := r.gates[params.ID]
	if !ok {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	now := time.Now().UTC()
	gate.Status = params.Status
	gate.ReviewedBy = params.ReviewedBy
	gate.ReviewNote = params.ReviewNote
	gate.ReviewedAt = now
	gate.UpdatedAt = now
	r.gates[gate.ID] = gate
	return gate, nil
}

func approvalGateMatches(gate domain.ApprovalGate, params SaveApprovalGateParams) bool {
	return gate.EpisodeID == params.EpisodeID &&
		gate.GateType == params.GateType &&
		gate.SubjectType == params.SubjectType &&
		gate.SubjectID == params.SubjectID
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
	if params.Analysis.WorkflowRunID != "" {
		if run, ok := r.runs[params.Analysis.WorkflowRunID]; ok {
			run.Status = domain.WorkflowRunStatusSucceeded
			run.UpdatedAt = job.UpdatedAt
			r.runs[run.ID] = run
		}
	}

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
		StorySourceID:   params.StorySourceID,
		WorkflowRunID:   params.WorkflowRunID,
		GenerationJobID: params.GenerationJobID,
		Version:         r.nextStoryAnalysisVersion(params.EpisodeID),
		Status:          params.Status,
		Summary:         params.Summary,
		Themes:          append([]string{}, params.Themes...),
		CharacterSeeds:  append([]string{}, params.CharacterSeeds...),
		SceneSeeds:      append([]string{}, params.SceneSeeds...),
		PropSeeds:       append([]string{}, params.PropSeeds...),
		Outline:         append([]domain.StoryBeat{}, params.Outline...),
		AgentOutputs:    append([]domain.StoryAgentOutput{}, params.AgentOutputs...),
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

func (r *MemoryProductionRepository) GetCharacter(_ context.Context, characterID string) (domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	character, ok := r.chars[characterID]
	if !ok {
		return domain.Character{}, domain.ErrNotFound
	}
	return character, nil
}

func (r *MemoryProductionRepository) SaveCharacterBible(
	_ context.Context,
	params SaveCharacterBibleParams,
) (domain.Character, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	character, ok := r.chars[params.CharacterID]
	if !ok {
		return domain.Character{}, domain.ErrNotFound
	}
	bible := params.CharacterBible
	character.CharacterBible = &bible
	character.UpdatedAt = time.Now().UTC()
	r.chars[character.ID] = character
	return character, nil
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

func (r *MemoryProductionRepository) GetStoryboardShot(
	_ context.Context,
	shotID string,
) (domain.StoryboardShot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	shot, ok := r.shots[shotID]
	if !ok {
		return domain.StoryboardShot{}, domain.ErrNotFound
	}
	return shot, nil
}

func (r *MemoryProductionRepository) SaveShotPromptPack(
	_ context.Context,
	params SaveShotPromptPackParams,
) (domain.ShotPromptPack, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	pack, ok := r.prompts[params.ShotID]
	if !ok {
		pack.ID = params.ID
		pack.CreatedAt = now
	}
	pack.ProjectID = params.ProjectID
	pack.EpisodeID = params.EpisodeID
	pack.ShotID = params.ShotID
	pack.Provider = params.Provider
	pack.Model = params.Model
	pack.Preset = params.Preset
	pack.TaskType = params.TaskType
	pack.DirectPrompt = params.DirectPrompt
	pack.NegativePrompt = params.NegativePrompt
	pack.TimeSlices = append([]domain.PromptTimeSlice{}, params.TimeSlices...)
	pack.ReferenceBindings = append([]domain.PromptReferenceBinding{}, params.ReferenceBindings...)
	pack.Params = cloneMap(params.Params)
	pack.UpdatedAt = now
	r.prompts[params.ShotID] = pack
	return pack, nil
}

func (r *MemoryProductionRepository) GetShotPromptPack(
	_ context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pack, ok := r.prompts[shotID]
	if !ok {
		return domain.ShotPromptPack{}, domain.ErrNotFound
	}
	return pack, nil
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
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

func (r *MemoryProductionRepository) CreateAsset(
	_ context.Context,
	params CreateAssetParams,
) (domain.Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.findOrCreateAssetLocked(params), nil
}

func (r *MemoryProductionRepository) findOrCreateAssetLocked(params CreateAssetParams) domain.Asset {
	for _, existing := range r.assets {
		if existing.EpisodeID == params.EpisodeID &&
			existing.Kind == params.Kind &&
			existing.Purpose == params.Purpose &&
			existing.URI == params.URI {
			return existing
		}
	}

	now := time.Now().UTC()
	asset := domain.Asset{
		ID: params.ID, ProjectID: params.ProjectID, EpisodeID: params.EpisodeID,
		Kind: params.Kind, Purpose: params.Purpose, URI: params.URI, Status: params.Status,
		CreatedAt: now, UpdatedAt: now,
	}
	r.assets[asset.ID] = asset
	return asset
}

func (r *MemoryProductionRepository) ListAssetsByEpisode(_ context.Context, episodeID string) ([]domain.Asset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	assets := make([]domain.Asset, 0)
	for _, asset := range r.assets {
		if asset.EpisodeID == episodeID {
			assets = append(assets, asset)
		}
	}
	sort.Slice(assets, func(i int, j int) bool {
		if assets[i].Kind == assets[j].Kind {
			return assets[i].Purpose < assets[j].Purpose
		}
		return assets[i].Kind < assets[j].Kind
	})
	return assets, nil
}

func (r *MemoryProductionRepository) GetAsset(_ context.Context, assetID string) (domain.Asset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	asset, ok := r.assets[assetID]
	if !ok {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, nil
}

func (r *MemoryProductionRepository) LockAsset(_ context.Context, assetID string) (domain.Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	asset, ok := r.assets[assetID]
	if !ok {
		return domain.Asset{}, domain.ErrNotFound
	}
	asset.Status = domain.AssetStatusReady
	asset.UpdatedAt = time.Now().UTC()
	r.assets[assetID] = asset
	return asset, nil
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

func (r *MemoryProductionRepository) GetTimelineByID(_ context.Context, timelineID string) (domain.Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, timeline := range r.timelines {
		if timeline.ID == timelineID {
			return timeline, nil
		}
	}
	return domain.Timeline{}, domain.ErrNotFound
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

func (r *MemoryProductionRepository) ListExportsByStatus(
	_ context.Context,
	status domain.ExportStatus,
	limit int,
) ([]domain.Export, error) {
	if limit <= 0 {
		return []domain.Export{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	exports := make([]domain.Export, 0)
	for _, item := range r.exports {
		if item.Status == status {
			exports = append(exports, item)
		}
	}
	sort.Slice(exports, func(i int, j int) bool {
		if exports[i].CreatedAt.Equal(exports[j].CreatedAt) {
			return exports[i].ID < exports[j].ID
		}
		return exports[i].CreatedAt.Before(exports[j].CreatedAt)
	})
	if len(exports) > limit {
		exports = exports[:limit]
	}
	return exports, nil
}

func (r *MemoryProductionRepository) AdvanceExportStatus(
	_ context.Context,
	params AdvanceExportStatusParams,
) (domain.Export, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	export, ok := r.exports[params.ID]
	if !ok || export.Status != params.From {
		return domain.Export{}, domain.ErrNotFound
	}
	export.Status = params.To
	export.UpdatedAt = time.Now().UTC()
	r.exports[export.ID] = export
	return export, nil
}
