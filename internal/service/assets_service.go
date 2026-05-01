package service

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// AssetRecoveryEvent is a synthesized timeline entry derived from
// asset.created_at / updated_at / status. We do not maintain an
// asset audit log, so events are virtual rather than persisted.
type AssetRecoveryEvent struct {
	Status    domain.AssetStatus
	Message   string
	CreatedAt time.Time
}

type AssetRecoverySummary struct {
	IsTerminal      bool
	IsRecoverable   bool
	IsLocked        bool
	CurrentStatus   domain.AssetStatus
	StatusEnteredAt time.Time
	LastEventAt     time.Time
	TotalEventCount int
	NextHint        string
}

type AssetRecovery struct {
	Asset   domain.Asset
	Events  []AssetRecoveryEvent
	Summary AssetRecoverySummary
}

// GetAssetRecovery returns a lightweight write-history view for an asset.
// Events are synthesized from asset timestamps; no new audit table is required.
func (s *ProductionService) GetAssetRecovery(ctx context.Context, assetID string) (AssetRecovery, error) {
	if strings.TrimSpace(assetID) == "" {
		return AssetRecovery{}, fmt.Errorf("%w: asset id is required", domain.ErrInvalidInput)
	}
	asset, err := s.production.GetAsset(ctx, assetID)
	if err != nil {
		return AssetRecovery{}, err
	}
	if err := s.authorizeScopedResource(ctx, asset.ProjectID, asset.EpisodeID); err != nil {
		return AssetRecovery{}, err
	}
	events := buildAssetRecoveryEvents(asset)
	return AssetRecovery{
		Asset:   asset,
		Events:  events,
		Summary: buildAssetRecoverySummary(asset, events),
	}, nil
}

func buildAssetRecoveryEvents(asset domain.Asset) []AssetRecoveryEvent {
	events := []AssetRecoveryEvent{
		{Status: domain.AssetStatusDraft, Message: "asset created", CreatedAt: asset.CreatedAt},
	}
	if asset.Status != domain.AssetStatusDraft && !asset.UpdatedAt.Equal(asset.CreatedAt) {
		msg := "current status"
		if asset.Status == domain.AssetStatusReady {
			msg = "asset locked (status=ready)"
		}
		events = append(events, AssetRecoveryEvent{
			Status:    asset.Status,
			Message:   msg,
			CreatedAt: asset.UpdatedAt,
		})
	}
	return events
}

func buildAssetRecoverySummary(asset domain.Asset, events []AssetRecoveryEvent) AssetRecoverySummary {
	summary := AssetRecoverySummary{
		CurrentStatus:   asset.Status,
		IsLocked:        asset.Status == domain.AssetStatusReady,
		IsTerminal:      isTerminalAssetStatus(asset.Status),
		IsRecoverable:   isRecoverableAssetStatus(asset.Status),
		TotalEventCount: len(events),
		StatusEnteredAt: asset.UpdatedAt,
		LastEventAt:     asset.UpdatedAt,
	}
	if summary.StatusEnteredAt.IsZero() {
		summary.StatusEnteredAt = asset.CreatedAt
	}
	if summary.LastEventAt.IsZero() {
		summary.LastEventAt = asset.CreatedAt
	}
	for _, ev := range events {
		if ev.CreatedAt.After(summary.LastEventAt) {
			summary.LastEventAt = ev.CreatedAt
		}
	}
	summary.NextHint = assetRecoveryNextHint(asset.Status)
	return summary
}

func isTerminalAssetStatus(status domain.AssetStatus) bool {
	switch status {
	case domain.AssetStatusReady, domain.AssetStatusFailed, domain.AssetStatusArchived:
		return true
	}
	return false
}

func isRecoverableAssetStatus(status domain.AssetStatus) bool {
	switch status {
	case domain.AssetStatusDraft, domain.AssetStatusGenerating, domain.AssetStatusFailed:
		return true
	}
	return false
}

func assetRecoveryNextHint(status domain.AssetStatus) string {
	switch status {
	case domain.AssetStatusDraft:
		return "asset is a draft candidate; lock to mark as the production reference"
	case domain.AssetStatusGenerating:
		return "asset generation is still in flight; check the underlying generation job"
	case domain.AssetStatusReady:
		return "asset is locked as the production reference"
	case domain.AssetStatusFailed:
		return "previous generation failed; retry generation or pick another candidate"
	case domain.AssetStatusArchived:
		return "asset is archived; restore via re-creation if needed"
	}
	return ""
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
