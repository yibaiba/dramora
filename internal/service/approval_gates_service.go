package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

const defaultApprovalReviewer = "studio"

type approvalGateSeed struct {
	gateType      string
	subjectType   string
	subjectID     string
	workflowRunID string
}

func (s *ProductionService) SeedEpisodeApprovalGates(
	ctx context.Context,
	episode domain.Episode,
) ([]domain.ApprovalGate, error) {
	seeds, err := s.approvalGateSeeds(ctx, episode)
	if err != nil {
		return nil, err
	}
	gates := make([]domain.ApprovalGate, 0, len(seeds))
	for _, seed := range seeds {
		gate, err := s.saveApprovalGate(ctx, episode, seed)
		if err != nil {
			return nil, err
		}
		gates = append(gates, gate)
	}
	return gates, nil
}

func (s *ProductionService) ListApprovalGates(
	ctx context.Context,
	episodeID string,
) ([]domain.ApprovalGate, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	return s.production.ListApprovalGates(ctx, episodeID)
}

func (s *ProductionService) ApproveApprovalGate(
	ctx context.Context,
	gateID string,
	reviewedBy string,
	reviewNote string,
) (domain.ApprovalGate, error) {
	return s.reviewApprovalGate(ctx, gateID, domain.ApprovalGateStatusApproved, reviewedBy, reviewNote)
}

func (s *ProductionService) RequestApprovalChanges(
	ctx context.Context,
	gateID string,
	reviewedBy string,
	reviewNote string,
) (domain.ApprovalGate, error) {
	return s.reviewApprovalGate(ctx, gateID, domain.ApprovalGateStatusChangesRequested, reviewedBy, reviewNote)
}

func (s *ProductionService) approvalGateSeeds(
	ctx context.Context,
	episode domain.Episode,
) ([]approvalGateSeed, error) {
	analysis, err := s.latestStoryAnalysis(ctx, episode.ID)
	if err != nil {
		return nil, err
	}
	seeds := []approvalGateSeed{{
		gateType: "story_direction", subjectType: "story_analysis",
		subjectID: analysis.ID, workflowRunID: analysis.WorkflowRunID,
	}}
	mapSeeds, err := s.storyMapApprovalSeeds(ctx, episode, analysis.WorkflowRunID)
	if err != nil {
		return nil, err
	}
	storyboardSeeds, err := s.storyboardApprovalSeed(ctx, episode, analysis.WorkflowRunID)
	if err != nil {
		return nil, err
	}
	timelineSeeds, err := s.timelineApprovalSeed(ctx, episode, analysis.WorkflowRunID)
	if err != nil {
		return nil, err
	}
	seeds = append(seeds, mapSeeds...)
	seeds = append(seeds, storyboardSeeds...)
	seeds = append(seeds, timelineSeeds...)
	return seeds, nil
}

func (s *ProductionService) storyMapApprovalSeeds(
	ctx context.Context,
	episode domain.Episode,
	workflowRunID string,
) ([]approvalGateSeed, error) {
	storyMap, err := s.production.GetStoryMap(ctx, episode.ID)
	if err != nil {
		return nil, err
	}
	seeds := make([]approvalGateSeed, 0, 3)
	if len(storyMap.Characters) > 0 {
		seeds = append(seeds, approvalGateSeed{"character_lock", "character_map", episode.ID, workflowRunID})
	}
	if len(storyMap.Scenes) > 0 {
		seeds = append(seeds, approvalGateSeed{"scene_lock", "scene_map", episode.ID, workflowRunID})
	}
	if len(storyMap.Props) > 0 {
		seeds = append(seeds, approvalGateSeed{"prop_lock", "prop_map", episode.ID, workflowRunID})
	}
	return seeds, nil
}

func (s *ProductionService) storyboardApprovalSeed(
	ctx context.Context,
	episode domain.Episode,
	workflowRunID string,
) ([]approvalGateSeed, error) {
	shots, err := s.production.ListStoryboardShots(ctx, episode.ID)
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		return nil, nil
	}
	return []approvalGateSeed{{"storyboard_approval", "storyboard", episode.ID, workflowRunID}}, nil
}

func (s *ProductionService) timelineApprovalSeed(
	ctx context.Context,
	episode domain.Episode,
	workflowRunID string,
) ([]approvalGateSeed, error) {
	timeline, err := s.production.GetEpisodeTimeline(ctx, episode.ID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []approvalGateSeed{{"final_timeline", "timeline", timeline.ID, workflowRunID}}, nil
}

func (s *ProductionService) saveApprovalGate(
	ctx context.Context,
	episode domain.Episode,
	seed approvalGateSeed,
) (domain.ApprovalGate, error) {
	id, err := domain.NewID()
	if err != nil {
		return domain.ApprovalGate{}, err
	}
	return s.production.SaveApprovalGate(ctx, repo.SaveApprovalGateParams{
		ID: id, ProjectID: episode.ProjectID, EpisodeID: episode.ID,
		WorkflowRunID: seed.workflowRunID, GateType: seed.gateType,
		SubjectType: seed.subjectType, SubjectID: seed.subjectID,
		Status: domain.ApprovalGateStatusPending,
	})
}

func (s *ProductionService) reviewApprovalGate(
	ctx context.Context,
	gateID string,
	status domain.ApprovalGateStatus,
	reviewedBy string,
	reviewNote string,
) (domain.ApprovalGate, error) {
	if strings.TrimSpace(gateID) == "" {
		return domain.ApprovalGate{}, fmt.Errorf("%w: approval gate id is required", domain.ErrInvalidInput)
	}
	gate, err := s.production.GetApprovalGate(ctx, gateID)
	if err != nil {
		return domain.ApprovalGate{}, err
	}
	if err := gate.Status.ValidateTransition(status); err != nil {
		return domain.ApprovalGate{}, err
	}
	return s.production.ReviewApprovalGate(ctx, repo.ReviewApprovalGateParams{
		ID: gateID, Status: status,
		ReviewedBy: defaultReviewer(reviewedBy), ReviewNote: strings.TrimSpace(reviewNote),
	})
}

func defaultReviewer(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultApprovalReviewer
	}
	return trimmed
}
