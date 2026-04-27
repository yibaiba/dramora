package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

func (s *ProductionService) completeGeneratedStoryAnalysis(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (domain.StoryAnalysis, error) {
	if err := generationJob.Status.ValidateTransition(domain.GenerationJobStatusSucceeded); err != nil {
		return domain.StoryAnalysis{}, err
	}
	source, err := s.latestStorySourceOrDefault(ctx, generationJob)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	analysisParams, err := generatedStoryAnalysisParams(generationJob, source)
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

func (s *ProductionService) latestStorySourceOrDefault(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (domain.StorySource, error) {
	source, err := s.production.LatestStorySource(ctx, generationJob.EpisodeID)
	if errors.Is(err, domain.ErrNotFound) {
		return defaultStorySource(generationJob), nil
	}
	return source, err
}

func defaultStorySource(generationJob domain.GenerationJob) domain.StorySource {
	return domain.StorySource{
		ProjectID:   generationJob.ProjectID,
		EpisodeID:   generationJob.EpisodeID,
		Title:       "默认故事样例",
		ContentText: "少年在云端宗门发现天门试炼的秘密。对立势力逼近，他必须在守护同伴和追寻身世之间做出选择。最终主角凭借关键线索完成试炼。",
		Language:    "zh-CN",
	}
}

func generatedStoryAnalysisParams(
	generationJob domain.GenerationJob,
	source domain.StorySource,
) (repo.CreateStoryAnalysisParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.CreateStoryAnalysisParams{}, err
	}
	analysis := analyzeStorySource(source)

	return repo.CreateStoryAnalysisParams{
		ID:              id,
		ProjectID:       generationJob.ProjectID,
		EpisodeID:       generationJob.EpisodeID,
		StorySourceID:   source.ID,
		WorkflowRunID:   generationJob.WorkflowRunID,
		GenerationJobID: generationJob.ID,
		Status:          domain.StoryAnalysisStatusGenerated,
		Summary:         analysis.summary,
		Themes:          analysis.themes,
		CharacterSeeds:  analysis.characterSeeds,
		SceneSeeds:      analysis.sceneSeeds,
		PropSeeds:       analysis.propSeeds,
		Outline:         analysis.outline,
		AgentOutputs:    analysis.agentOutputs,
	}, nil
}
