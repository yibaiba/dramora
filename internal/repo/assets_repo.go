package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) CreateAsset(
	ctx context.Context,
	params CreateAssetParams,
) (domain.Asset, error) {
	asset, err := scanAsset(r.pool.QueryRow(ctx, createAssetSQL,
		params.ID,
		params.ProjectID,
		params.EpisodeID,
		params.Kind,
		params.Purpose,
		params.URI,
		params.Status,
	))
	return asset, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) CompleteGenerationJobWithResult(
	ctx context.Context,
	params CompleteGenerationJobWithResultParams,
) (domain.GenerationJob, domain.Asset, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	defer tx.Rollback(ctx)

	asset, err := scanAsset(tx.QueryRow(ctx, createAssetSQL,
		params.Asset.ID,
		params.Asset.ProjectID,
		params.Asset.EpisodeID,
		params.Asset.Kind,
		params.Asset.Purpose,
		params.Asset.URI,
		params.Asset.Status,
	))
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, mapForeignKeyViolation(err)
	}
	jobParams := params.Job
	jobParams.ResultAssetID = asset.ID
	job, err := advanceGenerationJobStatusTx(ctx, tx, jobParams)
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	return job, asset, nil
}

func (r *PostgresProductionRepository) ListAssetsByEpisode(
	ctx context.Context,
	episodeID string,
) ([]domain.Asset, error) {
	rows, err := r.pool.Query(ctx, listEpisodeAssetsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (r *PostgresProductionRepository) GetAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	asset, err := scanAsset(r.pool.QueryRow(ctx, getAssetSQL, assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, err
}

func (r *PostgresProductionRepository) LockAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	asset, err := scanAsset(r.pool.QueryRow(ctx, lockAssetSQL, assetID, domain.AssetStatusReady))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, err
}
