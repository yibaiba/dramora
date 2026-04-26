package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

type ProductionRepository interface {
	CreateStoryAnalysisRun(ctx context.Context, params CreateStoryAnalysisRunParams) (StoryAnalysisRun, error)
	GetWorkflowRun(ctx context.Context, workflowRunID string) (domain.WorkflowRun, error)
	ListGenerationJobs(ctx context.Context) ([]domain.GenerationJob, error)
	ListGenerationJobsByStatus(ctx context.Context, status domain.GenerationJobStatus, limit int) ([]domain.GenerationJob, error)
	GetGenerationJob(ctx context.Context, generationJobID string) (domain.GenerationJob, error)
	AdvanceGenerationJobStatus(ctx context.Context, params AdvanceGenerationJobStatusParams) (domain.GenerationJob, error)
	GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error)
	SaveEpisodeTimeline(ctx context.Context, params SaveEpisodeTimelineParams) (domain.Timeline, error)
}

type CreateStoryAnalysisRunParams struct {
	WorkflowRunID   string
	GenerationJobID string
	ProjectID       string
	EpisodeID       string
	RequestKey      string
	Provider        string
	Model           string
	Prompt          string
}

type StoryAnalysisRun struct {
	WorkflowRun   domain.WorkflowRun
	GenerationJob domain.GenerationJob
}

type AdvanceGenerationJobStatusParams struct {
	ID           string
	From         domain.GenerationJobStatus
	To           domain.GenerationJobStatus
	EventMessage string
}

type SaveEpisodeTimelineParams struct {
	ID         string
	EpisodeID  string
	Status     domain.TimelineStatus
	DurationMS int
}

type PostgresProductionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProductionRepository(pool *pgxpool.Pool) *PostgresProductionRepository {
	return &PostgresProductionRepository{pool: pool}
}

func (r *PostgresProductionRepository) CreateStoryAnalysisRun(
	ctx context.Context,
	params CreateStoryAnalysisRunParams,
) (StoryAnalysisRun, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoryAnalysisRun{}, err
	}
	defer tx.Rollback(ctx)

	run, err := scanWorkflowRun(tx.QueryRow(ctx, createWorkflowRunSQL,
		params.WorkflowRunID,
		params.ProjectID,
		params.EpisodeID,
		domain.WorkflowRunStatusRunning,
	))
	if err != nil {
		return StoryAnalysisRun{}, err
	}

	job, err := scanGenerationJob(tx.QueryRow(ctx, createGenerationJobSQL,
		params.GenerationJobID,
		params.ProjectID,
		params.EpisodeID,
		params.WorkflowRunID,
		params.RequestKey,
		params.Provider,
		params.Model,
		"story_analysis",
		domain.GenerationJobStatusQueued,
		params.Prompt,
	))
	if err != nil {
		return StoryAnalysisRun{}, err
	}

	if _, err := tx.Exec(ctx, createGenerationJobEventSQL, params.GenerationJobID, job.Status, "story analysis queued"); err != nil {
		return StoryAnalysisRun{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoryAnalysisRun{}, err
	}

	return StoryAnalysisRun{WorkflowRun: run, GenerationJob: job}, nil
}

func (r *PostgresProductionRepository) GetWorkflowRun(
	ctx context.Context,
	workflowRunID string,
) (domain.WorkflowRun, error) {
	run, err := scanWorkflowRun(r.pool.QueryRow(ctx, getWorkflowRunSQL, workflowRunID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkflowRun{}, domain.ErrNotFound
	}
	return run, err
}

func (r *PostgresProductionRepository) ListGenerationJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	rows, err := r.pool.Query(ctx, listGenerationJobsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanGenerationJobs(rows)
}

func (r *PostgresProductionRepository) ListGenerationJobsByStatus(
	ctx context.Context,
	status domain.GenerationJobStatus,
	limit int,
) ([]domain.GenerationJob, error) {
	rows, err := r.pool.Query(ctx, listGenerationJobsByStatusSQL, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanGenerationJobs(rows)
}

func (r *PostgresProductionRepository) GetGenerationJob(
	ctx context.Context,
	generationJobID string,
) (domain.GenerationJob, error) {
	job, err := scanGenerationJob(r.pool.QueryRow(ctx, getGenerationJobSQL, generationJobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	return job, err
}

func (r *PostgresProductionRepository) AdvanceGenerationJobStatus(
	ctx context.Context,
	params AdvanceGenerationJobStatusParams,
) (domain.GenerationJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	defer tx.Rollback(ctx)

	job, err := scanGenerationJob(tx.QueryRow(ctx, advanceGenerationJobStatusSQL, params.ID, params.From, params.To))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if _, err := tx.Exec(ctx, createGenerationJobEventSQL, params.ID, params.To, params.EventMessage); err != nil {
		return domain.GenerationJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func (r *PostgresProductionRepository) GetEpisodeTimeline(
	ctx context.Context,
	episodeID string,
) (domain.Timeline, error) {
	timeline, err := scanTimeline(r.pool.QueryRow(ctx, getEpisodeTimelineSQL, episodeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Timeline{}, domain.ErrNotFound
	}
	return timeline, err
}

func (r *PostgresProductionRepository) SaveEpisodeTimeline(
	ctx context.Context,
	params SaveEpisodeTimelineParams,
) (domain.Timeline, error) {
	timeline, err := scanTimeline(r.pool.QueryRow(ctx, saveEpisodeTimelineSQL,
		params.ID,
		params.EpisodeID,
		params.Status,
		params.DurationMS,
	))
	if isForeignKeyViolation(err) {
		return domain.Timeline{}, domain.ErrNotFound
	}
	return timeline, err
}
