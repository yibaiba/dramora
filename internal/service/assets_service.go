package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

func (s *ProductionService) SeedEpisodeAssets(ctx context.Context, episode domain.Episode) ([]domain.Asset, error) {
	storyMap, err := s.production.GetStoryMap(ctx, episode.ID)
	if err != nil {
		return nil, err
	}

	params, err := assetCandidateParams(episode, storyMap)
	if err != nil {
		return nil, err
	}
	assets := make([]domain.Asset, 0, len(params))
	for _, item := range params {
		asset, err := s.production.CreateAsset(ctx, item)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func (s *ProductionService) ListEpisodeAssets(ctx context.Context, episodeID string) ([]domain.Asset, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	if err := s.authorizeEpisode(ctx, episodeID); err != nil {
		return nil, err
	}
	return s.production.ListAssetsByEpisode(ctx, episodeID)
}

func (s *ProductionService) LockAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	if strings.TrimSpace(assetID) == "" {
		return domain.Asset{}, fmt.Errorf("%w: asset id is required", domain.ErrInvalidInput)
	}
	asset, err := s.production.GetAsset(ctx, assetID)
	if err != nil {
		return domain.Asset{}, err
	}
	if err := s.authorizeScopedResource(ctx, asset.ProjectID, asset.EpisodeID); err != nil {
		return domain.Asset{}, err
	}
	return s.production.LockAsset(ctx, assetID)
}

func assetCandidateParams(episode domain.Episode, storyMap repo.StoryMap) ([]repo.CreateAssetParams, error) {
	total := len(storyMap.Characters) + len(storyMap.Scenes) + len(storyMap.Props)
	params := make([]repo.CreateAssetParams, 0, total)
	characters, err := characterAssetParams(episode, storyMap.Characters)
	if err != nil {
		return nil, err
	}
	scenes, err := sceneAssetParams(episode, storyMap.Scenes)
	if err != nil {
		return nil, err
	}
	props, err := propAssetParams(episode, storyMap.Props)
	if err != nil {
		return nil, err
	}
	params = append(params, characters...)
	params = append(params, scenes...)
	params = append(params, props...)
	return params, nil
}

func characterAssetParams(episode domain.Episode, items []domain.Character) ([]repo.CreateAssetParams, error) {
	params := make([]repo.CreateAssetParams, 0, len(items))
	for _, item := range items {
		param, err := assetParam(episode, "character", item.Code)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func sceneAssetParams(episode domain.Episode, items []domain.Scene) ([]repo.CreateAssetParams, error) {
	params := make([]repo.CreateAssetParams, 0, len(items))
	for _, item := range items {
		param, err := assetParam(episode, "scene", item.Code)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func propAssetParams(episode domain.Episode, items []domain.Prop) ([]repo.CreateAssetParams, error) {
	params := make([]repo.CreateAssetParams, 0, len(items))
	for _, item := range items {
		param, err := assetParam(episode, "prop", item.Code)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func assetParam(episode domain.Episode, kind string, code string) (repo.CreateAssetParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.CreateAssetParams{}, err
	}
	return repo.CreateAssetParams{
		ID:        id,
		ProjectID: episode.ProjectID,
		EpisodeID: episode.ID,
		Kind:      kind,
		Purpose:   code,
		URI:       fmt.Sprintf("manmu://episodes/%s/%s/%s/candidate-1", episode.ID, kind, code),
		Status:    domain.AssetStatusDraft,
	}, nil
}
