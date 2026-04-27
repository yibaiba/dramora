package repo

import (
	"context"
	"encoding/json"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) CreateGenerationJob(
	ctx context.Context,
	params CreateGenerationJobParams,
) (domain.GenerationJob, error) {
	payload, err := json.Marshal(params.Params)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	defer tx.Rollback(ctx)

	job, err := scanGenerationJob(tx.QueryRow(ctx, createGenerationJobWithParamsSQL,
		params.ID, params.ProjectID, params.EpisodeID, nullableUUID(params.WorkflowRunID),
		params.RequestKey, params.Provider, params.Model, params.TaskType, params.Status,
		params.Prompt, payload,
	))
	if err != nil {
		return domain.GenerationJob{}, mapForeignKeyViolation(err)
	}
	if job.ID != params.ID {
		if err := tx.Commit(ctx); err != nil {
			return domain.GenerationJob{}, err
		}
		return job, nil
	}
	if _, err := tx.Exec(ctx, createGenerationJobEventSQL, job.ID, job.Status, params.EventMessage); err != nil {
		return domain.GenerationJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}
