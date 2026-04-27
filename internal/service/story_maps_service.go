package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

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
