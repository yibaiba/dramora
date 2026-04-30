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

func (r *PostgresProductionRepository) ListGenerationJobEvents(
	ctx context.Context,
	generationJobID string,
	limit int,
) ([]domain.GenerationJobEvent, error) {
	rows, err := r.pool.Query(ctx, listGenerationJobEventsSQL, generationJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.GenerationJobEvent, 0)
	for rows.Next() {
		var (
			ev     domain.GenerationJobEvent
			status string
		)
		if err := rows.Scan(&ev.ID, &ev.GenerationJobID, &status, &ev.Message, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Status = domain.GenerationJobStatus(status)
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}
