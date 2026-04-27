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

func (r *PostgresProductionRepository) LockAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	asset, err := scanAsset(r.pool.QueryRow(ctx, lockAssetSQL, assetID, domain.AssetStatusReady))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, err
}
