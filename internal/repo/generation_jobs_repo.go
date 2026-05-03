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

// RetryJob creates a new job copying the original job's parameters
func (r *PostgresProductionRepository) RetryJob(ctx context.Context, originalJobID string) (domain.GenerationJob, error) {
	newJobID, err := domain.NewID()
	if err != nil {
		return domain.GenerationJob{}, err
	}
	requestKey := "retry-" + originalJobID

	job, err := scanGenerationJob(r.pool.QueryRow(ctx, retryGenerationJobSQL, newJobID, requestKey, originalJobID))
	if err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

// UpdateJobPriority updates a job's priority value
func (r *PostgresProductionRepository) UpdateJobPriority(ctx context.Context, jobID string, priority int) error {
	job, err := scanGenerationJob(r.pool.QueryRow(ctx, updateGenerationJobPrioritySQL, jobID, priority))
	if err != nil {
		return err
	}
	if job.ID == "" {
		return domain.ErrNotFound
	}
	return nil
}

// ListJobsByPriority returns jobs sorted by priority DESC for a given episode and status
func (r *PostgresProductionRepository) ListJobsByPriority(ctx context.Context, episodeID string, status domain.GenerationJobStatus) ([]domain.GenerationJob, error) {
	rows, err := r.pool.Query(ctx, listGenerationJobsByPrioritySQL, episodeID, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGenerationJobs(rows)
}

// QueueStatus represents the pause/resume state of a queue
type QueueStatus struct {
	EpisodeID string
	Paused    bool
}

// PauseEpisodeQueue pauses job processing for an episode
func (r *PostgresProductionRepository) PauseEpisodeQueue(ctx context.Context, episodeID string) error {
	_, err := r.pool.Exec(ctx, upsertQueueStatusSQL, episodeID, true)
	return err
}

// ResumeEpisodeQueue resumes job processing for an episode
func (r *PostgresProductionRepository) ResumeEpisodeQueue(ctx context.Context, episodeID string) error {
	_, err := r.pool.Exec(ctx, upsertQueueStatusSQL, episodeID, false)
	return err
}

// IsEpisodeQueuePaused returns whether an episode's queue is paused
func (r *PostgresProductionRepository) IsEpisodeQueuePaused(ctx context.Context, episodeID string) (bool, error) {
	var paused bool
	var pausedAt, updatedAt interface{}
	err := r.pool.QueryRow(ctx, getQueueStatusSQL, episodeID).Scan(nil, &paused, &pausedAt, &updatedAt)
	if err != nil {
		// If queue_status doesn't exist, it's not paused by default
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}
	return paused, nil
}
