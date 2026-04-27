package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

const (
	sd2DefaultNegativePrompt = "no flicker, no jitter, no face distortion, no extra fingers, no costume drift, no unreadable text"
	maxSD2References         = 9
)

func (s *ProductionService) GenerateShotPromptPack(
	ctx context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	shot, err := s.production.GetStoryboardShot(ctx, shotID)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	assets, err := s.production.ListAssetsByEpisode(ctx, shot.EpisodeID)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	packParams, err := buildShotPromptPackParams(shot, readyReferenceAssets(assets))
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	return s.production.SaveShotPromptPack(ctx, packParams)
}

func (s *ProductionService) GetShotPromptPack(
	ctx context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	return s.production.GetShotPromptPack(ctx, shotID)
}

func buildShotPromptPackParams(
	shot domain.StoryboardShot,
	refs []domain.Asset,
) (repo.SaveShotPromptPackParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.SaveShotPromptPackParams{}, err
	}
	profile := provider.SeedanceFastProfile()
	bindings := promptReferenceBindings(refs)
	durationSec := promptDurationSeconds(shot.DurationMS)
	taskType := provider.TaskTypeTextToVideo
	if len(bindings) > 0 {
		taskType = provider.TaskTypeImageToVideo
	}
	return repo.SaveShotPromptPackParams{
		ID: id, ProjectID: shot.ProjectID, EpisodeID: shot.EpisodeID, ShotID: shot.ID,
		Provider: profile.Provider, Model: profile.Model, Preset: profile.Preset,
		TaskType: string(taskType), DirectPrompt: directSD2Prompt(shot, bindings),
		NegativePrompt:    sd2DefaultNegativePrompt,
		TimeSlices:        promptTimeSlices(shot),
		ReferenceBindings: bindings,
		Params:            promptPackParams(profile, taskType, durationSec, bindings),
	}, nil
}

func readyReferenceAssets(assets []domain.Asset) []domain.Asset {
	refs := make([]domain.Asset, 0, len(assets))
	for _, asset := range assets {
		if asset.Status == domain.AssetStatusReady {
			refs = append(refs, asset)
		}
	}
	sort.Slice(refs, func(i int, j int) bool {
		if refs[i].Kind == refs[j].Kind {
			return refs[i].URI < refs[j].URI
		}
		return refs[i].Kind < refs[j].Kind
	})
	if len(refs) > maxSD2References {
		return refs[:maxSD2References]
	}
	return refs
}

func promptReferenceBindings(refs []domain.Asset) []domain.PromptReferenceBinding {
	bindings := make([]domain.PromptReferenceBinding, 0, len(refs))
	for index, asset := range refs {
		role := "reference_image"
		if index == 0 {
			role = "first_frame"
		}
		bindings = append(bindings, domain.PromptReferenceBinding{
			Token: fmt.Sprintf("@image%d", index+1), Role: role,
			AssetID: asset.ID, Kind: asset.Kind, URI: asset.URI,
		})
	}
	return bindings
}

func promptDurationSeconds(durationMS int) int {
	if durationMS <= 0 {
		return provider.SeedanceFastProfile().DefaultDurationSec
	}
	seconds := (durationMS + 999) / 1000
	if seconds > 15 {
		return 15
	}
	return seconds
}

func directSD2Prompt(shot domain.StoryboardShot, refs []domain.PromptReferenceBinding) string {
	parts := []string{
		fmt.Sprintf("%s %s: %s", shot.Code, shot.Title, shot.Description),
		shot.Prompt,
		"AI manju cinematic panel, expressive composition, consistent character design, coherent scene lighting.",
		"Camera: slow push-in with stable framing; Style: high quality animated comic video.",
		"Stability: smooth motion, stable face, consistent outfit, normal anatomy, no flicker.",
	}
	if len(refs) > 0 {
		parts = append(parts, "Use locked references "+referenceTokenList(refs)+"; keep @image2 and later refs as continuity anchors.")
	}
	return strings.Join(nonEmptyStrings(parts), " ")
}

func promptTimeSlices(shot domain.StoryboardShot) []domain.PromptTimeSlice {
	duration := shot.DurationMS
	if duration <= 0 {
		duration = provider.SeedanceFastProfile().DefaultDurationSec * 1000
	}
	firstEnd := duration / 3
	secondEnd := (duration * 2) / 3
	return []domain.PromptTimeSlice{
		{StartMS: 0, EndMS: firstEnd, Prompt: "Establish the subject and scene mood.", CameraWork: "slow push-in", ShotSize: "wide", VisualFocus: shot.Title},
		{StartMS: firstEnd, EndMS: secondEnd, Prompt: "Advance the key action with clear motion.", CameraWork: "gentle tracking", ShotSize: "medium", VisualFocus: shot.Description},
		{StartMS: secondEnd, EndMS: duration, Prompt: "Hold a clean final pose for continuity.", CameraWork: "stable hold", ShotSize: "close", VisualFocus: "last frame continuity"},
	}
}

func promptPackParams(
	profile provider.ModelProfile,
	taskType provider.TaskType,
	durationSec int,
	refs []domain.PromptReferenceBinding,
) map[string]any {
	return map[string]any{
		"ratio":             profile.DefaultRatio,
		"resolution":        profile.DefaultResolution,
		"duration":          durationSec,
		"service_tier":      profile.ServiceTier,
		"return_last_frame": true,
		"watermark":         false,
		"provider_mode":     "fake_default_real_when_ark_key_present",
		"task_type":         string(taskType),
		"reference_count":   len(refs),
	}
}

func referenceTokenList(refs []domain.PromptReferenceBinding) string {
	tokens := make([]string, 0, len(refs))
	for _, ref := range refs {
		tokens = append(tokens, ref.Token)
	}
	return strings.Join(tokens, ", ")
}

func nonEmptyStrings(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
