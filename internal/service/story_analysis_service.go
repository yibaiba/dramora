package service

import (
	"context"
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
