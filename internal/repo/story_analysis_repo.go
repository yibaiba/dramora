package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

type storyAnalysisQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *PostgresProductionRepository) CompleteStoryAnalysisJob(
	ctx context.Context,
	params CompleteStoryAnalysisJobParams,
) (StoryAnalysisCompletion, error) {
	payloads, err := storyAnalysisPayloads(params.Analysis)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	defer tx.Rollback(ctx)

	job, err := advanceGenerationJobStatusTx(ctx, tx, params.Job)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}

	analysis, err := createStoryAnalysisWithPayloads(ctx, tx, params.Analysis, payloads)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoryAnalysisCompletion{}, err
	}
	return StoryAnalysisCompletion{GenerationJob: job, StoryAnalysis: analysis}, nil
}

func (r *PostgresProductionRepository) CreateStoryAnalysis(
	ctx context.Context,
	params CreateStoryAnalysisParams,
) (domain.StoryAnalysis, error) {
	payloads, err := storyAnalysisPayloads(params)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}

	analysis, err := createStoryAnalysisWithPayloads(ctx, r.pool, params, payloads)
	return analysis, err
}

func (r *PostgresProductionRepository) ListStoryAnalyses(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryAnalysis, error) {
	rows, err := r.pool.Query(ctx, listStoryAnalysesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanStoryAnalyses(rows)
}

func (r *PostgresProductionRepository) GetStoryAnalysis(
	ctx context.Context,
	analysisID string,
) (domain.StoryAnalysis, error) {
	analysis, err := scanStoryAnalysis(r.pool.QueryRow(ctx, getStoryAnalysisSQL, analysisID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analysis, err
}

func createStoryAnalysisWithPayloads(
	ctx context.Context,
	queryer storyAnalysisQueryer,
	params CreateStoryAnalysisParams,
	payloads [6]string,
) (domain.StoryAnalysis, error) {
	analysis, err := scanStoryAnalysis(queryer.QueryRow(ctx, createStoryAnalysisSQL,
		params.ID,
		params.ProjectID,
		params.EpisodeID,
		nullableUUID(params.StorySourceID),
		nullableUUID(params.WorkflowRunID),
		nullableUUID(params.GenerationJobID),
		params.Status,
		params.Summary,
		payloads[0],
		payloads[1],
		payloads[2],
		payloads[3],
		payloads[4],
		payloads[5],
	))
	if isForeignKeyViolation(err) {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analysis, err
}

func storyAnalysisPayloads(params CreateStoryAnalysisParams) ([6]string, error) {
	var payloads [6]string
	for index, value := range []any{
		params.Themes, params.CharacterSeeds, params.SceneSeeds,
		params.PropSeeds, params.Outline, params.AgentOutputs,
	} {
		payload, err := json.Marshal(value)
		if err != nil {
			return [6]string{}, err
		}
		payloads[index] = string(payload)
	}
	return payloads, nil
}
