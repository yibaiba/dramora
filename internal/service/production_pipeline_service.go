package service

import (
	"context"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type SeedEpisodeProductionResult struct {
	StoryMap        repo.StoryMap
	Assets          []domain.Asset
	StoryboardShots []domain.StoryboardShot
	ApprovalGates   []domain.ApprovalGate
}

func (s *ProductionService) SeedEpisodeProduction(
	ctx context.Context,
	episode domain.Episode,
) (SeedEpisodeProductionResult, error) {
	storyMap, err := s.SeedStoryMap(ctx, episode)
	if err != nil {
		return SeedEpisodeProductionResult{}, err
	}
	assets, err := s.SeedEpisodeAssets(ctx, episode)
	if err != nil {
		return SeedEpisodeProductionResult{}, err
	}
	shots, err := s.SeedStoryboardShots(ctx, episode)
	if err != nil {
		return SeedEpisodeProductionResult{}, err
	}
	gates, err := s.SeedEpisodeApprovalGates(ctx, episode)
	if err != nil {
		return SeedEpisodeProductionResult{}, err
	}
	return SeedEpisodeProductionResult{
		StoryMap:        storyMap,
		Assets:          assets,
		StoryboardShots: shots,
		ApprovalGates:   gates,
	}, nil
}
