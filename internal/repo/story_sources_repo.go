package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) CreateStorySource(
	ctx context.Context,
	params CreateStorySourceParams,
) (domain.StorySource, error) {
	source, err := scanStorySource(r.pool.QueryRow(ctx, createStorySourceSQL,
		params.ID,
		params.ProjectID,
		params.EpisodeID,
		params.SourceType,
		params.Title,
		params.ContentText,
		params.Language,
	))
	return source, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) ListStorySources(
	ctx context.Context,
	episodeID string,
) ([]domain.StorySource, error) {
	rows, err := r.pool.Query(ctx, listStorySourcesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStorySources(rows)
}

func (r *PostgresProductionRepository) LatestStorySource(
	ctx context.Context,
	episodeID string,
) (domain.StorySource, error) {
	source, err := scanStorySource(r.pool.QueryRow(ctx, latestStorySourceSQL, episodeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StorySource{}, domain.ErrNotFound
	}
	return source, err
}
