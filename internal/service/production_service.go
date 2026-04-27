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

func (s *ProductionService) completeGeneratedStoryAnalysis(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (domain.StoryAnalysis, error) {
	if err := generationJob.Status.ValidateTransition(domain.GenerationJobStatusSucceeded); err != nil {
		return domain.StoryAnalysis{}, err
	}
	analysisParams, err := generatedStoryAnalysisParams(generationJob)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}

	completion, err := s.production.CompleteStoryAnalysisJob(ctx, repo.CompleteStoryAnalysisJobParams{
		Job: repo.AdvanceGenerationJobStatusParams{
			ID:           generationJob.ID,
			From:         generationJob.Status,
			To:           domain.GenerationJobStatusSucceeded,
			EventMessage: "no-op worker completed story analysis and wrote artifact",
		},
		Analysis: analysisParams,
	})
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	return completion.StoryAnalysis, nil
}

func generatedStoryAnalysisParams(generationJob domain.GenerationJob) (repo.CreateStoryAnalysisParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.CreateStoryAnalysisParams{}, err
	}

	return repo.CreateStoryAnalysisParams{
		ID:              id,
		ProjectID:       generationJob.ProjectID,
		EpisodeID:       generationJob.EpisodeID,
		WorkflowRunID:   generationJob.WorkflowRunID,
		GenerationJobID: generationJob.ID,
		Status:          domain.StoryAnalysisStatusGenerated,
		Summary:         "No-op story analyst extracted MVP seeds for character, scene, prop, and beat planning.",
		Themes:          []string{"identity", "choice", "visual contrast"},
		CharacterSeeds:  []string{"C01 protagonist", "C02 opposing force"},
		SceneSeeds:      []string{"S01 opening scene", "S02 conflict scene", "S03 resolution scene"},
		PropSeeds:       []string{"P01 signature item", "P02 story clue"},
	}, nil
}

func (s *ProductionService) GetGenerationJob(ctx context.Context, id string) (domain.GenerationJob, error) {
	if strings.TrimSpace(id) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id is required", domain.ErrInvalidInput)
	}
	return s.production.GetGenerationJob(ctx, id)
}

func (s *ProductionService) ListStoryAnalyses(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryAnalysis, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.ListStoryAnalyses(ctx, episodeID)
}

func (s *ProductionService) GetStoryAnalysis(ctx context.Context, id string) (domain.StoryAnalysis, error) {
	if strings.TrimSpace(id) == "" {
		return domain.StoryAnalysis{}, fmt.Errorf("%w: story analysis id is required", domain.ErrInvalidInput)
	}
	return s.production.GetStoryAnalysis(ctx, id)
}

func (s *ProductionService) SeedStoryMap(ctx context.Context, episode domain.Episode) (repo.StoryMap, error) {
	analysis, err := s.latestStoryAnalysis(ctx, episode.ID)
	if err != nil {
		return repo.StoryMap{}, err
	}
	params, err := storyMapSeedParams(episode, analysis)
	if err != nil {
		return repo.StoryMap{}, err
	}
	return s.production.SaveStoryMap(ctx, params)
}

func (s *ProductionService) GetStoryMap(ctx context.Context, episodeID string) (repo.StoryMap, error) {
	if strings.TrimSpace(episodeID) == "" {
		return repo.StoryMap{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.GetStoryMap(ctx, episodeID)
}

func (s *ProductionService) SeedStoryboardShots(
	ctx context.Context,
	episode domain.Episode,
) ([]domain.StoryboardShot, error) {
	analysis, err := s.latestStoryAnalysis(ctx, episode.ID)
	if err != nil {
		return nil, err
	}
	storyMap, err := s.production.GetStoryMap(ctx, episode.ID)
	if err != nil {
		return nil, err
	}
	params, err := storyboardSeedParams(episode, analysis, storyMap.Scenes)
	if err != nil {
		return nil, err
	}
	return s.production.SaveStoryboardShots(ctx, params)
}

func (s *ProductionService) ListStoryboardShots(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryboardShot, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.ListStoryboardShots(ctx, episodeID)
}

func (s *ProductionService) GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error) {
	if strings.TrimSpace(episodeID) == "" {
		return domain.Timeline{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.GetEpisodeTimeline(ctx, episodeID)
}

func (s *ProductionService) SaveEpisodeTimeline(
	ctx context.Context,
	input SaveTimelineInput,
) (domain.Timeline, error) {
	if strings.TrimSpace(input.EpisodeID) == "" {
		return domain.Timeline{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	if input.DurationMS < 0 {
		return domain.Timeline{}, fmt.Errorf("%w: duration_ms must be non-negative", domain.ErrInvalidInput)
	}

	id, err := domain.NewID()
	if err != nil {
		return domain.Timeline{}, err
	}

	if len(input.Tracks) == 0 {
		return s.production.SaveEpisodeTimeline(ctx, repo.SaveEpisodeTimelineParams{
			ID:         id,
			EpisodeID:  input.EpisodeID,
			Status:     domain.TimelineStatusSaved,
			DurationMS: input.DurationMS,
		})
	}

	tracks, err := timelineTrackParams(input.Tracks)
	if err != nil {
		return domain.Timeline{}, err
	}
	return s.production.SaveEpisodeTimelineGraph(ctx, repo.SaveEpisodeTimelineGraphParams{
		ID:         id,
		EpisodeID:  input.EpisodeID,
		Status:     domain.TimelineStatusSaved,
		DurationMS: input.DurationMS,
		Tracks:     tracks,
	})
}

func (s *ProductionService) StartEpisodeExport(ctx context.Context, episodeID string) (domain.Export, error) {
	timeline, err := s.GetEpisodeTimeline(ctx, episodeID)
	if err != nil {
		return domain.Export{}, err
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.Export{}, err
	}
	return s.production.CreateExport(ctx, repo.CreateExportParams{
		ID:         id,
		TimelineID: timeline.ID,
		Status:     domain.ExportStatusQueued,
		Format:     "mp4",
	})
}

func (s *ProductionService) GetExport(ctx context.Context, id string) (domain.Export, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Export{}, fmt.Errorf("%w: export id is required", domain.ErrInvalidInput)
	}
	return s.production.GetExport(ctx, id)
}

func (s *ProductionService) latestStoryAnalysis(ctx context.Context, episodeID string) (domain.StoryAnalysis, error) {
	analyses, err := s.ListStoryAnalyses(ctx, episodeID)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	if len(analyses) == 0 {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analyses[0], nil
}

func storyMapSeedParams(
	episode domain.Episode,
	analysis domain.StoryAnalysis,
) (repo.SaveStoryMapParams, error) {
	characters, err := storyMapItemParams(episode, analysis.ID, "C", analysis.CharacterSeeds)
	if err != nil {
		return repo.SaveStoryMapParams{}, err
	}
	scenes, err := storyMapItemParams(episode, analysis.ID, "S", analysis.SceneSeeds)
	if err != nil {
		return repo.SaveStoryMapParams{}, err
	}
	props, err := storyMapItemParams(episode, analysis.ID, "P", analysis.PropSeeds)
	if err != nil {
		return repo.SaveStoryMapParams{}, err
	}
	return repo.SaveStoryMapParams{Characters: characters, Scenes: scenes, Props: props}, nil
}

func storyMapItemParams(
	episode domain.Episode,
	analysisID string,
	prefix string,
	seeds []string,
) ([]repo.SaveStoryMapItemParams, error) {
	items := make([]repo.SaveStoryMapItemParams, 0, len(seeds))
	for index, seed := range seeds {
		id, err := domain.NewID()
		if err != nil {
			return nil, err
		}
		code := fmt.Sprintf("%s%02d", prefix, index+1)
		items = append(items, repo.SaveStoryMapItemParams{
			ID: id, ProjectID: episode.ProjectID, EpisodeID: episode.ID,
			StoryAnalysisID: analysisID, Code: code, Name: seed, Description: seed,
		})
	}
	return items, nil
}

func storyboardSeedParams(
	episode domain.Episode,
	analysis domain.StoryAnalysis,
	scenes []domain.Scene,
) (repo.SaveStoryboardShotsParams, error) {
	shotCount := len(scenes)
	if shotCount == 0 {
		shotCount = 3
	}
	shots := make([]repo.SaveStoryboardShotParams, 0, shotCount)
	for index := 0; index < shotCount; index++ {
		shot, err := storyboardShotParam(episode, analysis, scenes, index)
		if err != nil {
			return repo.SaveStoryboardShotsParams{}, err
		}
		shots = append(shots, shot)
	}
	return repo.SaveStoryboardShotsParams{Shots: shots}, nil
}

func storyboardShotParam(
	episode domain.Episode,
	analysis domain.StoryAnalysis,
	scenes []domain.Scene,
	index int,
) (repo.SaveStoryboardShotParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.SaveStoryboardShotParams{}, err
	}
	code := fmt.Sprintf("SH%03d", index+1)
	sceneID := ""
	title := fmt.Sprintf("Shot %d", index+1)
	if index < len(scenes) {
		sceneID = scenes[index].ID
		title = scenes[index].Name
	}
	return repo.SaveStoryboardShotParams{
		ID: id, ProjectID: episode.ProjectID, EpisodeID: episode.ID,
		StoryAnalysisID: analysis.ID, SceneID: sceneID, Code: code, Title: title,
		Description: "Seeded shot card from story analysis and scene map.",
		Prompt:      "Cinematic manju panel, consistent character and scene continuity.",
		Position:    index + 1, DurationMS: 3000,
	}, nil
}

func timelineTrackParams(
	inputs []SaveTimelineTrackInput,
) ([]repo.SaveTimelineTrackParams, error) {
	tracks := make([]repo.SaveTimelineTrackParams, 0, len(inputs))
	for _, input := range inputs {
		id, err := domain.NewID()
		if err != nil {
			return nil, err
		}
		clips, err := timelineClipParams(input.Clips)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, repo.SaveTimelineTrackParams{
			ID: id, Kind: input.Kind, Name: input.Name, Position: input.Position, Clips: clips,
		})
	}
	return tracks, nil
}

func timelineClipParams(inputs []SaveTimelineClipInput) ([]repo.SaveTimelineClipParams, error) {
	clips := make([]repo.SaveTimelineClipParams, 0, len(inputs))
	for _, input := range inputs {
		id, err := domain.NewID()
		if err != nil {
			return nil, err
		}
		clips = append(clips, repo.SaveTimelineClipParams{
			ID: id, AssetID: input.AssetID, Kind: input.Kind, StartMS: input.StartMS,
			DurationMS: input.DurationMS, TrimStartMS: input.TrimStartMS,
		})
	}
	return clips, nil
}
