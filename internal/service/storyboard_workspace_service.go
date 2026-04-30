package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type StoryboardWorkspace struct {
	EpisodeID       string
	Summary         StoryboardWorkspaceSummary
	StoryMap        repo.StoryMap
	StoryboardShots []StoryboardWorkspaceShot
	Assets          []domain.Asset
	ApprovalGates   []domain.ApprovalGate
	GenerationJobs  []domain.GenerationJob
}

type StoryboardWorkspaceSummary struct {
	AnalysisCount             int
	StoryMapReady             bool
	ReadyAssetsCount          int
	PendingApprovalGatesCount int
}

type StoryboardWorkspaceShot struct {
	Shot                domain.StoryboardShot
	Scene               *domain.Scene
	PromptPack          *domain.ShotPromptPack
	LatestGenerationJob *domain.GenerationJob
}

func (s *ProductionService) GetStoryboardWorkspace(
	ctx context.Context,
	episodeID string,
) (StoryboardWorkspace, error) {
	if strings.TrimSpace(episodeID) == "" {
		return StoryboardWorkspace{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}

	analyses, err := s.ListStoryAnalyses(ctx, episodeID)
	if err != nil {
		return StoryboardWorkspace{}, err
	}
	storyMap, err := s.GetStoryMap(ctx, episodeID)
	if err != nil {
		return StoryboardWorkspace{}, err
	}
	shots, err := s.ListStoryboardShots(ctx, episodeID)
	if err != nil {
		return StoryboardWorkspace{}, err
	}
	assets, err := s.ListEpisodeAssets(ctx, episodeID)
	if err != nil {
		return StoryboardWorkspace{}, err
	}
	gates, err := s.ListApprovalGates(ctx, episodeID)
	if err != nil {
		return StoryboardWorkspace{}, err
	}
	jobs, err := s.ListGenerationJobs(ctx)
	if err != nil {
		return StoryboardWorkspace{}, err
	}

	episodeJobs := filterGenerationJobsByEpisode(jobs, episodeID)
	sceneByID := make(map[string]domain.Scene, len(storyMap.Scenes))
	for _, scene := range storyMap.Scenes {
		sceneByID[scene.ID] = scene
	}
	latestJobsByShot := latestGenerationJobByShot(episodeJobs)

	workspaceShots := make([]StoryboardWorkspaceShot, 0, len(shots))
	for _, shot := range shots {
		workspaceShot := StoryboardWorkspaceShot{Shot: shot}
		if scene, ok := sceneByID[shot.SceneID]; ok {
			sceneCopy := scene
			workspaceShot.Scene = &sceneCopy
		}

		pack, packErr := s.production.GetShotPromptPack(ctx, shot.ID)
		switch {
		case packErr == nil:
			packCopy := pack
			workspaceShot.PromptPack = &packCopy
		case errors.Is(packErr, domain.ErrNotFound):
		default:
			return StoryboardWorkspace{}, packErr
		}

		if latestJob, ok := latestJobsByShot[shot.ID]; ok {
			jobCopy := latestJob
			workspaceShot.LatestGenerationJob = &jobCopy
		}
		workspaceShots = append(workspaceShots, workspaceShot)
	}

	return StoryboardWorkspace{
		EpisodeID: episodeID,
		Summary: StoryboardWorkspaceSummary{
			AnalysisCount:             len(analyses),
			StoryMapReady:             len(storyMap.Characters)+len(storyMap.Scenes)+len(storyMap.Props) > 0,
			ReadyAssetsCount:          countReadyAssets(assets),
			PendingApprovalGatesCount: countPendingApprovalGates(gates),
		},
		StoryMap:        storyMap,
		StoryboardShots: workspaceShots,
		Assets:          assets,
		ApprovalGates:   gates,
		GenerationJobs:  episodeJobs,
	}, nil
}

func filterGenerationJobsByEpisode(jobs []domain.GenerationJob, episodeID string) []domain.GenerationJob {
	filtered := make([]domain.GenerationJob, 0, len(jobs))
	for _, job := range jobs {
		if job.EpisodeID == episodeID {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func latestGenerationJobByShot(jobs []domain.GenerationJob) map[string]domain.GenerationJob {
	latest := make(map[string]domain.GenerationJob)
	for _, job := range jobs {
		shotID, _ := job.Params["shot_id"].(string)
		if shotID == "" {
			continue
		}
		if _, exists := latest[shotID]; exists {
			continue
		}
		latest[shotID] = job
	}
	return latest
}

func countReadyAssets(assets []domain.Asset) int {
	count := 0
	for _, asset := range assets {
		if asset.Status == domain.AssetStatusReady {
			count++
		}
	}
	return count
}

func countPendingApprovalGates(gates []domain.ApprovalGate) int {
	count := 0
	for _, gate := range gates {
		if gate.Status == domain.ApprovalGateStatusPending || gate.Status == domain.ApprovalGateStatusChangesRequested {
			count++
		}
	}
	return count
}
