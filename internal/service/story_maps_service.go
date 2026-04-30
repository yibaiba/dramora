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
	if err := s.authorizeEpisode(ctx, episodeID); err != nil {
		return repo.StoryMap{}, err
	}
	return s.production.GetStoryMap(ctx, episodeID)
}

type SaveCharacterBibleInput struct {
	CharacterBible domain.CharacterBible
}

func (s *ProductionService) SaveCharacterBible(
	ctx context.Context,
	characterID string,
	input SaveCharacterBibleInput,
) (domain.Character, error) {
	if strings.TrimSpace(characterID) == "" {
		return domain.Character{}, fmt.Errorf("%w: character id is required", domain.ErrInvalidInput)
	}
	input.CharacterBible = normalizeCharacterBible(input.CharacterBible)
	if input.CharacterBible.Anchor == "" {
		return domain.Character{}, fmt.Errorf("%w: character bible anchor is required", domain.ErrInvalidInput)
	}
	character, err := s.production.GetCharacter(ctx, characterID)
	if err != nil {
		return domain.Character{}, err
	}
	if err := s.authorizeScopedResource(ctx, character.ProjectID, character.EpisodeID); err != nil {
		return domain.Character{}, err
	}
	if err := s.validateCharacterBibleReferenceAssets(ctx, character, input.CharacterBible); err != nil {
		return domain.Character{}, err
	}
	return s.production.SaveCharacterBible(ctx, repo.SaveCharacterBibleParams{
		CharacterID:    characterID,
		CharacterBible: input.CharacterBible,
	})
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
	if err := s.authorizeEpisode(ctx, episodeID); err != nil {
		return nil, err
	}
	return s.production.ListStoryboardShots(ctx, episodeID)
}

type UpdateStoryboardShotInput struct {
	Title       string
	Description string
	Prompt      string
	DurationMS  int
}

func (s *ProductionService) UpdateStoryboardShot(
	ctx context.Context,
	shotID string,
	input UpdateStoryboardShotInput,
) (domain.StoryboardShot, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.StoryboardShot{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.StoryboardShot{}, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return domain.StoryboardShot{}, fmt.Errorf("%w: prompt is required", domain.ErrInvalidInput)
	}
	if input.DurationMS <= 0 {
		return domain.StoryboardShot{}, fmt.Errorf("%w: duration must be positive", domain.ErrInvalidInput)
	}
	shot, err := s.production.GetStoryboardShot(ctx, shotID)
	if err != nil {
		return domain.StoryboardShot{}, err
	}
	if err := s.authorizeScopedResource(ctx, shot.ProjectID, shot.EpisodeID); err != nil {
		return domain.StoryboardShot{}, err
	}
	shots, err := s.production.SaveStoryboardShots(ctx, repo.SaveStoryboardShotsParams{
		Shots: []repo.SaveStoryboardShotParams{{
			ID:              shot.ID,
			ProjectID:       shot.ProjectID,
			EpisodeID:       shot.EpisodeID,
			StoryAnalysisID: shot.StoryAnalysisID,
			SceneID:         shot.SceneID,
			Code:            shot.Code,
			Title:           strings.TrimSpace(input.Title),
			Description:     strings.TrimSpace(input.Description),
			Prompt:          strings.TrimSpace(input.Prompt),
			Position:        shot.Position,
			DurationMS:      input.DurationMS,
		}},
	})
	if err != nil {
		return domain.StoryboardShot{}, err
	}
	return shots[0], nil
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

func normalizeCharacterBible(input domain.CharacterBible) domain.CharacterBible {
	input.Anchor = strings.TrimSpace(input.Anchor)
	input.Wardrobe = strings.TrimSpace(input.Wardrobe)
	input.Notes = strings.TrimSpace(input.Notes)
	input.Palette.Skin = strings.TrimSpace(input.Palette.Skin)
	input.Palette.Hair = strings.TrimSpace(input.Palette.Hair)
	input.Palette.Accent = strings.TrimSpace(input.Palette.Accent)
	input.Palette.Eyes = strings.TrimSpace(input.Palette.Eyes)
	input.Palette.Costume = strings.TrimSpace(input.Palette.Costume)
	input.Expressions = compactUniqueStrings(input.Expressions)
	input.ReferenceAngles = compactUniqueStrings(input.ReferenceAngles)
	input.ReferenceAssets = normalizeCharacterBibleReferenceAssets(input.ReferenceAssets)
	return input
}

func normalizeCharacterBibleReferenceAssets(
	items []domain.CharacterBibleReferenceAsset,
) []domain.CharacterBibleReferenceAsset {
	result := make([]domain.CharacterBibleReferenceAsset, 0, len(items))
	for _, item := range items {
		normalized := domain.CharacterBibleReferenceAsset{
			Angle:   strings.TrimSpace(item.Angle),
			AssetID: strings.TrimSpace(item.AssetID),
		}
		if normalized.Angle == "" && normalized.AssetID == "" {
			continue
		}
		result = append(result, normalized)
	}
	return result
}

func compactUniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *ProductionService) validateCharacterBibleReferenceAssets(
	ctx context.Context,
	character domain.Character,
	bible domain.CharacterBible,
) error {
	if len(bible.ReferenceAssets) == 0 {
		return nil
	}

	allowedAngles := make(map[string]struct{}, len(bible.ReferenceAngles))
	for _, angle := range bible.ReferenceAngles {
		allowedAngles[angle] = struct{}{}
	}

	assets, err := s.production.ListAssetsByEpisode(ctx, character.EpisodeID)
	if err != nil {
		return err
	}
	assetsByID := make(map[string]domain.Asset, len(assets))
	for _, asset := range assets {
		assetsByID[asset.ID] = asset
	}

	seenAngles := make(map[string]struct{}, len(bible.ReferenceAssets))
	for _, reference := range bible.ReferenceAssets {
		if reference.Angle == "" {
			return fmt.Errorf("%w: reference asset angle is required", domain.ErrInvalidInput)
		}
		if reference.AssetID == "" {
			return fmt.Errorf("%w: reference asset id is required", domain.ErrInvalidInput)
		}
		if _, ok := allowedAngles[reference.Angle]; !ok {
			return fmt.Errorf("%w: reference asset angle %q is not enabled", domain.ErrInvalidInput, reference.Angle)
		}
		if _, ok := seenAngles[reference.Angle]; ok {
			return fmt.Errorf("%w: duplicate reference asset angle %q", domain.ErrInvalidInput, reference.Angle)
		}
		seenAngles[reference.Angle] = struct{}{}

		asset, ok := assetsByID[reference.AssetID]
		if !ok {
			return fmt.Errorf("%w: reference asset %q was not found in this episode", domain.ErrInvalidInput, reference.AssetID)
		}
		if asset.Status != domain.AssetStatusReady {
			return fmt.Errorf("%w: reference asset %q must be locked before assignment", domain.ErrInvalidInput, reference.AssetID)
		}
		if asset.Kind != "character" {
			return fmt.Errorf("%w: reference asset %q must be a character asset", domain.ErrInvalidInput, reference.AssetID)
		}
		if asset.Purpose != character.Code {
			return fmt.Errorf("%w: reference asset %q does not belong to character %s", domain.ErrInvalidInput, reference.AssetID, character.Code)
		}
	}
	return nil
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
