package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) SaveStoryMap(
	ctx context.Context,
	params SaveStoryMapParams,
) (StoryMap, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoryMap{}, err
	}
	defer tx.Rollback(ctx)

	storyMap, err := saveStoryMapTx(ctx, tx, params)
	if err != nil {
		return StoryMap{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoryMap{}, err
	}
	return storyMap, nil
}

func (r *PostgresProductionRepository) GetStoryMap(ctx context.Context, episodeID string) (StoryMap, error) {
	characters, err := r.listCharacters(ctx, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	scenes, err := r.listScenes(ctx, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	props, err := r.listProps(ctx, episodeID)
	return StoryMap{Characters: characters, Scenes: scenes, Props: props}, err
}

func (r *PostgresProductionRepository) SaveStoryboardShots(
	ctx context.Context,
	params SaveStoryboardShotsParams,
) ([]domain.StoryboardShot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	shots := make([]domain.StoryboardShot, 0, len(params.Shots))
	for _, shotParams := range params.Shots {
		shot, err := scanStoryboardShot(tx.QueryRow(ctx, upsertStoryboardShotSQL,
			shotParams.ID, shotParams.ProjectID, shotParams.EpisodeID,
			nullableUUID(shotParams.StoryAnalysisID), nullableUUID(shotParams.SceneID),
			shotParams.Code, shotParams.Title, shotParams.Description, shotParams.Prompt,
			shotParams.Position, shotParams.DurationMS,
		))
		if err != nil {
			return nil, mapForeignKeyViolation(err)
		}
		shots = append(shots, shot)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return shots, nil
}

func (r *PostgresProductionRepository) ListStoryboardShots(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryboardShot, error) {
	rows, err := r.pool.Query(ctx, listStoryboardShotsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStoryboardShots(rows)
}

func (r *PostgresProductionRepository) GetStoryboardShot(
	ctx context.Context,
	shotID string,
) (domain.StoryboardShot, error) {
	shot, err := scanStoryboardShot(r.pool.QueryRow(ctx, getStoryboardShotSQL, shotID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoryboardShot{}, domain.ErrNotFound
	}
	return shot, err
}

func saveStoryMapTx(ctx context.Context, tx pgx.Tx, params SaveStoryMapParams) (StoryMap, error) {
	storyMap := StoryMap{
		Characters: make([]domain.Character, 0, len(params.Characters)),
		Scenes:     make([]domain.Scene, 0, len(params.Scenes)),
		Props:      make([]domain.Prop, 0, len(params.Props)),
	}
	for _, item := range params.Characters {
		value, err := scanCharacter(tx.QueryRow(ctx, upsertCharacterSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Characters = append(storyMap.Characters, value)
	}
	for _, item := range params.Scenes {
		value, err := scanScene(tx.QueryRow(ctx, upsertSceneSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Scenes = append(storyMap.Scenes, value)
	}
	for _, item := range params.Props {
		value, err := scanProp(tx.QueryRow(ctx, upsertPropSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Props = append(storyMap.Props, value)
	}
	return storyMap, nil
}

func (r *PostgresProductionRepository) listCharacters(ctx context.Context, episodeID string) ([]domain.Character, error) {
	rows, err := r.pool.Query(ctx, listCharactersSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCharacters(rows)
}

func (r *PostgresProductionRepository) listScenes(ctx context.Context, episodeID string) ([]domain.Scene, error) {
	rows, err := r.pool.Query(ctx, listScenesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScenes(rows)
}

func (r *PostgresProductionRepository) listProps(ctx context.Context, episodeID string) ([]domain.Prop, error) {
	rows, err := r.pool.Query(ctx, listPropsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProps(rows)
}
