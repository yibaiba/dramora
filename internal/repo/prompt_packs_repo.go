package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) SaveShotPromptPack(
	ctx context.Context,
	params SaveShotPromptPackParams,
) (domain.ShotPromptPack, error) {
	timeSlices, err := json.Marshal(params.TimeSlices)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	references, err := json.Marshal(params.ReferenceBindings)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	promptParams, err := json.Marshal(params.Params)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	pack, err := scanShotPromptPack(r.pool.QueryRow(ctx, upsertShotPromptPackSQL,
		params.ID, params.ProjectID, params.EpisodeID, params.ShotID,
		params.Provider, params.Model, params.Preset, params.TaskType,
		params.DirectPrompt, params.NegativePrompt, timeSlices, references, promptParams,
	))
	return pack, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) GetShotPromptPack(
	ctx context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	pack, err := scanShotPromptPack(r.pool.QueryRow(ctx, getShotPromptPackSQL, shotID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ShotPromptPack{}, domain.ErrNotFound
	}
	return pack, err
}
