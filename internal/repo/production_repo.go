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
	CreateGenerationJob(ctx context.Context, params CreateGenerationJobParams) (domain.GenerationJob, error)
	AdvanceGenerationJobStatus(ctx context.Context, params AdvanceGenerationJobStatusParams) (domain.GenerationJob, error)
	ListApprovalGates(ctx context.Context, episodeID string) ([]domain.ApprovalGate, error)
	GetApprovalGate(ctx context.Context, gateID string) (domain.ApprovalGate, error)
	SaveApprovalGate(ctx context.Context, params SaveApprovalGateParams) (domain.ApprovalGate, error)
	ReviewApprovalGate(ctx context.Context, params ReviewApprovalGateParams) (domain.ApprovalGate, error)
	CompleteStoryAnalysisJob(ctx context.Context, params CompleteStoryAnalysisJobParams) (StoryAnalysisCompletion, error)
	CreateStoryAnalysis(ctx context.Context, params CreateStoryAnalysisParams) (domain.StoryAnalysis, error)
	ListStoryAnalyses(ctx context.Context, episodeID string) ([]domain.StoryAnalysis, error)
	GetStoryAnalysis(ctx context.Context, analysisID string) (domain.StoryAnalysis, error)
	SaveStoryMap(ctx context.Context, params SaveStoryMapParams) (StoryMap, error)
	GetStoryMap(ctx context.Context, episodeID string) (StoryMap, error)
	SaveStoryboardShots(ctx context.Context, params SaveStoryboardShotsParams) ([]domain.StoryboardShot, error)
	ListStoryboardShots(ctx context.Context, episodeID string) ([]domain.StoryboardShot, error)
	GetStoryboardShot(ctx context.Context, shotID string) (domain.StoryboardShot, error)
	SaveShotPromptPack(ctx context.Context, params SaveShotPromptPackParams) (domain.ShotPromptPack, error)
	GetShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error)
	CreateAsset(ctx context.Context, params CreateAssetParams) (domain.Asset, error)
	ListAssetsByEpisode(ctx context.Context, episodeID string) ([]domain.Asset, error)
	LockAsset(ctx context.Context, assetID string) (domain.Asset, error)
	GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error)
	SaveEpisodeTimeline(ctx context.Context, params SaveEpisodeTimelineParams) (domain.Timeline, error)
	SaveEpisodeTimelineGraph(ctx context.Context, params SaveEpisodeTimelineGraphParams) (domain.Timeline, error)
	CreateExport(ctx context.Context, params CreateExportParams) (domain.Export, error)
	GetExport(ctx context.Context, exportID string) (domain.Export, error)
	ListExportsByStatus(ctx context.Context, status domain.ExportStatus, limit int) ([]domain.Export, error)
	AdvanceExportStatus(ctx context.Context, params AdvanceExportStatusParams) (domain.Export, error)
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

type CreateGenerationJobParams struct {
	ID            string
	ProjectID     string
	EpisodeID     string
	WorkflowRunID string
	RequestKey    string
	Provider      string
	Model         string
	TaskType      string
	Status        domain.GenerationJobStatus
	Prompt        string
	Params        map[string]any
	EventMessage  string
}

type AdvanceGenerationJobStatusParams struct {
	ID             string
	From           domain.GenerationJobStatus
	To             domain.GenerationJobStatus
	ProviderTaskID string
	EventMessage   string
}

type SaveApprovalGateParams struct {
	ID            string
	ProjectID     string
	EpisodeID     string
	WorkflowRunID string
	GateType      string
	SubjectType   string
	SubjectID     string
	Status        domain.ApprovalGateStatus
}

type ReviewApprovalGateParams struct {
	ID         string
	Status     domain.ApprovalGateStatus
	ReviewedBy string
	ReviewNote string
}

type CreateStoryAnalysisParams struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	WorkflowRunID   string
	GenerationJobID string
	Status          domain.StoryAnalysisStatus
	Summary         string
	Themes          []string
	CharacterSeeds  []string
	SceneSeeds      []string
	PropSeeds       []string
}

type CompleteStoryAnalysisJobParams struct {
	Job      AdvanceGenerationJobStatusParams
	Analysis CreateStoryAnalysisParams
}

type StoryAnalysisCompletion struct {
	GenerationJob domain.GenerationJob
	StoryAnalysis domain.StoryAnalysis
}

type StoryMap struct {
	Characters []domain.Character
	Scenes     []domain.Scene
	Props      []domain.Prop
}

type SaveStoryMapParams struct {
	Characters []SaveStoryMapItemParams
	Scenes     []SaveStoryMapItemParams
	Props      []SaveStoryMapItemParams
}

type SaveStoryMapItemParams struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	Code            string
	Name            string
	Description     string
}

type SaveStoryboardShotsParams struct {
	Shots []SaveStoryboardShotParams
}

type SaveStoryboardShotParams struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	SceneID         string
	Code            string
	Title           string
	Description     string
	Prompt          string
	Position        int
	DurationMS      int
}

type CreateAssetParams struct {
	ID        string
	ProjectID string
	EpisodeID string
	Kind      string
	Purpose   string
	URI       string
	Status    domain.AssetStatus
}

type SaveShotPromptPackParams struct {
	ID                string
	ProjectID         string
	EpisodeID         string
	ShotID            string
	Provider          string
	Model             string
	Preset            string
	TaskType          string
	DirectPrompt      string
	NegativePrompt    string
	TimeSlices        []domain.PromptTimeSlice
	ReferenceBindings []domain.PromptReferenceBinding
	Params            map[string]any
}

type SaveEpisodeTimelineParams struct {
	ID         string
	EpisodeID  string
	Status     domain.TimelineStatus
	DurationMS int
}

type SaveEpisodeTimelineGraphParams struct {
	ID         string
	EpisodeID  string
	Status     domain.TimelineStatus
	DurationMS int
	Tracks     []SaveTimelineTrackParams
}

type SaveTimelineTrackParams struct {
	ID       string
	Kind     string
	Name     string
	Position int
	Clips    []SaveTimelineClipParams
}

type SaveTimelineClipParams struct {
	ID          string
	AssetID     string
	Kind        string
	StartMS     int
	DurationMS  int
	TrimStartMS int
}

type CreateExportParams struct {
	ID         string
	TimelineID string
	Status     domain.ExportStatus
	Format     string
}

type AdvanceExportStatusParams struct {
	ID   string
	From domain.ExportStatus
	To   domain.ExportStatus
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

	job, err := advanceGenerationJobStatusTx(ctx, tx, params)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func advanceGenerationJobStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdvanceGenerationJobStatusParams,
) (domain.GenerationJob, error) {
	job, err := scanGenerationJob(tx.QueryRow(
		ctx,
		advanceGenerationJobStatusSQL,
		params.ID,
		params.From,
		params.To,
		params.ProviderTaskID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if _, err := tx.Exec(ctx, createGenerationJobEventSQL, params.ID, params.To, params.EventMessage); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mapForeignKeyViolation(err error) error {
	if isForeignKeyViolation(err) {
		return domain.ErrNotFound
	}
	return err
}
