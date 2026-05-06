package repo

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/yibaiba/dramora/internal/domain"
)

type SQLiteProductionRepository struct {
	db *sql.DB
}

func NewSQLiteProductionRepository(db *sql.DB) *SQLiteProductionRepository {
	return &SQLiteProductionRepository{db: db}
}

func (r *SQLiteProductionRepository) CreateStorySource(ctx context.Context, params CreateStorySourceParams) (domain.StorySource, error) {
	_, err := r.db.ExecContext(ctx, sqliteCreateStorySourceSQL,
		params.ID, params.ProjectID, params.EpisodeID, params.SourceType, params.Title, params.ContentText, params.Language,
	)
	if err != nil {
		return domain.StorySource{}, sqliteMapFK(err)
	}
	return r.getStorySourceByID(ctx, params.ID)
}

func (r *SQLiteProductionRepository) ListStorySources(ctx context.Context, episodeID string) ([]domain.StorySource, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListStorySourcesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStorySources(rows)
}

func (r *SQLiteProductionRepository) LatestStorySource(ctx context.Context, episodeID string) (domain.StorySource, error) {
	row := r.db.QueryRowContext(ctx, sqliteLatestStorySourceSQL, episodeID)
	source, err := scanStorySource(row)
	if err == sql.ErrNoRows {
		return domain.StorySource{}, domain.ErrNotFound
	}
	return source, err
}

func (r *SQLiteProductionRepository) CreateStoryAnalysisRun(ctx context.Context, params CreateStoryAnalysisRunParams) (StoryAnalysisRun, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StoryAnalysisRun{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, sqliteCreateWorkflowRunSQL,
		params.WorkflowRunID, params.ProjectID, params.EpisodeID, domain.WorkflowRunStatusRunning,
	)
	if err != nil {
		return StoryAnalysisRun{}, err
	}
	run, err := scanSQLiteWorkflowRun(tx.QueryRowContext(ctx, sqliteGetWorkflowRunSQL, params.WorkflowRunID))
	if err != nil {
		return StoryAnalysisRun{}, err
	}

	_, err = tx.ExecContext(ctx, sqliteCreateGenerationJobSQL,
		params.GenerationJobID, params.ProjectID, params.EpisodeID, params.WorkflowRunID,
		params.RequestKey, params.Provider, params.Model, "story_analysis", domain.GenerationJobStatusQueued, params.Prompt,
	)
	if err != nil {
		return StoryAnalysisRun{}, err
	}
	job, err := scanSQLiteGenerationJob(tx.QueryRowContext(ctx, sqliteGetGenerationJobSQL, params.GenerationJobID))
	if err != nil {
		return StoryAnalysisRun{}, err
	}

	_, err = tx.ExecContext(ctx, sqliteCreateGenerationJobEventSQL, params.GenerationJobID, job.Status, "story analysis queued")
	if err != nil {
		return StoryAnalysisRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return StoryAnalysisRun{}, err
	}
	return StoryAnalysisRun{WorkflowRun: run, GenerationJob: job}, nil
}

func (r *SQLiteProductionRepository) GetWorkflowRun(ctx context.Context, workflowRunID string) (domain.WorkflowRun, error) {
	run, err := scanSQLiteWorkflowRun(r.db.QueryRowContext(ctx, sqliteGetWorkflowRunSQL, workflowRunID))
	if err == sql.ErrNoRows {
		return domain.WorkflowRun{}, domain.ErrNotFound
	}
	return run, err
}

func (r *SQLiteProductionRepository) SaveWorkflowCheckpoint(ctx context.Context, workflowRunID string, payload []byte) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	res, err := r.db.ExecContext(ctx, sqliteSaveWorkflowCheckpointSQL, string(payload), workflowRunID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteProductionRepository) LoadWorkflowCheckpoint(ctx context.Context, workflowRunID string) ([]byte, error) {
	var payload string
	if err := r.db.QueryRowContext(ctx, sqliteLoadWorkflowCheckpointSQL, workflowRunID).Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return []byte(payload), nil
}

func (r *SQLiteProductionRepository) ListGenerationJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListGenerationJobsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteGenerationJobs(rows)
}

func (r *SQLiteProductionRepository) ListGenerationJobsByStatus(ctx context.Context, status domain.GenerationJobStatus, limit int) ([]domain.GenerationJob, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListGenerationJobsByStatusSQL, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteGenerationJobs(rows)
}

func (r *SQLiteProductionRepository) GetGenerationJob(ctx context.Context, generationJobID string) (domain.GenerationJob, error) {
	job, err := scanSQLiteGenerationJob(r.db.QueryRowContext(ctx, sqliteGetGenerationJobSQL, generationJobID))
	if err == sql.ErrNoRows {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	return job, err
}

func (r *SQLiteProductionRepository) CreateGenerationJob(ctx context.Context, params CreateGenerationJobParams) (domain.GenerationJob, error) {
	payload, err := json.Marshal(params.Params)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, sqliteCreateGenerationJobWithParamsSQL,
		params.ID, params.ProjectID, params.EpisodeID, sqliteNullable(params.WorkflowRunID),
		params.RequestKey, params.Provider, params.Model, params.TaskType, params.Status,
		params.Prompt, string(payload),
	)
	if err != nil {
		return domain.GenerationJob{}, sqliteMapFK(err)
	}

	job, err := scanSQLiteGenerationJob(tx.QueryRowContext(ctx, sqliteGetGenerationJobByRequestKeySQL, params.RequestKey))
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if job.ID != params.ID {
		if err := tx.Commit(); err != nil {
			return domain.GenerationJob{}, err
		}
		return job, nil
	}
	_, err = tx.ExecContext(ctx, sqliteCreateGenerationJobEventSQL, job.ID, job.Status, params.EventMessage)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func (r *SQLiteProductionRepository) AdvanceGenerationJobStatus(ctx context.Context, params AdvanceGenerationJobStatusParams) (domain.GenerationJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	defer tx.Rollback()

	job, err := sqliteAdvanceJobTx(ctx, tx, params)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func sqliteAdvanceJobTx(ctx context.Context, tx *sql.Tx, params AdvanceGenerationJobStatusParams) (domain.GenerationJob, error) {
	res, err := tx.ExecContext(ctx, sqliteAdvanceGenerationJobStatusSQL,
		params.To, params.ProviderTaskID, params.ResultAssetID, params.ID, params.From,
	)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.GenerationJob{}, domain.ErrNotFound
	}
	_, err = tx.ExecContext(ctx, sqliteCreateGenerationJobEventSQL, params.ID, params.To, params.EventMessage)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	return scanSQLiteGenerationJob(tx.QueryRowContext(ctx, sqliteGetGenerationJobSQL, params.ID))
}

func scanSQLiteWorkflowRun(row rowScanner) (domain.WorkflowRun, error) {
	var (
		run       domain.WorkflowRun
		createdAt string
		updatedAt string
	)
	if err := row.Scan(
		&run.ID,
		&run.ProjectID,
		&run.EpisodeID,
		&run.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.WorkflowRun{}, err
	}
	var err error
	if run.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.WorkflowRun{}, err
	}
	if run.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.WorkflowRun{}, err
	}
	return run, nil
}

func scanSQLiteGenerationJobs(rows *sql.Rows) ([]domain.GenerationJob, error) {
	jobs := make([]domain.GenerationJob, 0)
	for rows.Next() {
		job, err := scanSQLiteGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func scanSQLiteGenerationJob(row rowScanner) (domain.GenerationJob, error) {
	var (
		job            domain.GenerationJob
		paramsPayload  []byte
		providerTaskID string
		resultAssetID  string
		createdAt      string
		updatedAt      string
	)
	if err := row.Scan(
		&job.ID,
		&job.ProjectID,
		&job.EpisodeID,
		&job.WorkflowRunID,
		&job.Provider,
		&job.Model,
		&job.TaskType,
		&job.Status,
		&job.Prompt,
		&paramsPayload,
		&providerTaskID,
		&resultAssetID,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.GenerationJob{}, err
	}
	if len(paramsPayload) > 0 {
		if err := json.Unmarshal(paramsPayload, &job.Params); err != nil {
			return domain.GenerationJob{}, err
		}
	}
	if job.Params == nil {
		job.Params = map[string]any{}
	}
	job.ProviderTaskID = providerTaskID
	job.ResultAssetID = resultAssetID
	var err error
	if job.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.GenerationJob{}, err
	}
	if job.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func (r *SQLiteProductionRepository) CompleteGenerationJobWithResult(ctx context.Context, params CompleteGenerationJobWithResultParams) (domain.GenerationJob, domain.Asset, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	defer tx.Rollback()

	asset, err := sqliteCreateAssetTx(ctx, tx, params.Asset)
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	jobParams := params.Job
	jobParams.ResultAssetID = asset.ID
	job, err := sqliteAdvanceJobTx(ctx, tx, jobParams)
	if err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.GenerationJob{}, domain.Asset{}, err
	}
	return job, asset, nil
}

func (r *SQLiteProductionRepository) ListGenerationJobEvents(
	ctx context.Context,
	generationJobID string,
	limit int,
) ([]domain.GenerationJobEvent, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListGenerationJobEventsSQL, generationJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.GenerationJobEvent, 0)
	for rows.Next() {
		var (
			ev          domain.GenerationJobEvent
			status      string
			messageNull sql.NullString
		)
		if err := rows.Scan(&ev.ID, &ev.GenerationJobID, &status, &messageNull, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Status = domain.GenerationJobStatus(status)
		ev.Message = messageNull.String
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

func (r *SQLiteProductionRepository) ListApprovalGates(ctx context.Context, episodeID string) ([]domain.ApprovalGate, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListApprovalGatesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteApprovalGates(rows)
}

func (r *SQLiteProductionRepository) GetApprovalGate(ctx context.Context, gateID string) (domain.ApprovalGate, error) {
	gate, err := scanSQLiteApprovalGate(r.db.QueryRowContext(ctx, sqliteGetApprovalGateSQL, gateID))
	if err == sql.ErrNoRows {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return gate, err
}

func (r *SQLiteProductionRepository) SaveApprovalGate(ctx context.Context, params SaveApprovalGateParams) (domain.ApprovalGate, error) {
	_, err := r.db.ExecContext(ctx, sqliteUpsertApprovalGateSQL,
		params.ID, params.ProjectID, params.EpisodeID, sqliteNullable(params.WorkflowRunID),
		params.GateType, params.SubjectType, params.SubjectID, params.Status,
	)
	if err != nil {
		return domain.ApprovalGate{}, sqliteMapFK(err)
	}
	gate, err := scanSQLiteApprovalGate(r.db.QueryRowContext(
		ctx,
		sqliteGetApprovalGateByKeySQL,
		params.EpisodeID,
		params.GateType,
		params.SubjectType,
		params.SubjectID,
	))
	if err == sql.ErrNoRows {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return gate, err
}

func (r *SQLiteProductionRepository) ReviewApprovalGate(ctx context.Context, params ReviewApprovalGateParams) (domain.ApprovalGate, error) {
	res, err := r.db.ExecContext(ctx, sqliteReviewApprovalGateSQL,
		params.Status, params.ReviewedBy, params.ReviewNote, params.ID,
	)
	if err != nil {
		return domain.ApprovalGate{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return r.GetApprovalGate(ctx, params.ID)
}

func (r *SQLiteProductionRepository) CompleteStoryAnalysisJob(ctx context.Context, params CompleteStoryAnalysisJobParams) (StoryAnalysisCompletion, error) {
	payloads, err := storyAnalysisPayloads(params.Analysis)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	defer tx.Rollback()

	job, err := sqliteAdvanceJobTx(ctx, tx, params.Job)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	analysis, err := sqliteCreateStoryAnalysisTx(ctx, tx, params.Analysis, payloads)
	if err != nil {
		return StoryAnalysisCompletion{}, err
	}
	if params.Analysis.WorkflowRunID != "" {
		if _, err := tx.ExecContext(ctx, sqliteCompleteWorkflowRunSQL, domain.WorkflowRunStatusSucceeded, params.Analysis.WorkflowRunID); err != nil {
			return StoryAnalysisCompletion{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return StoryAnalysisCompletion{}, err
	}
	return StoryAnalysisCompletion{GenerationJob: job, StoryAnalysis: analysis}, nil
}

func (r *SQLiteProductionRepository) CreateStoryAnalysis(ctx context.Context, params CreateStoryAnalysisParams) (domain.StoryAnalysis, error) {
	payloads, err := storyAnalysisPayloads(params)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	defer tx.Rollback()

	analysis, err := sqliteCreateStoryAnalysisTx(ctx, tx, params, payloads)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.StoryAnalysis{}, err
	}
	return analysis, nil
}

func sqliteCreateStoryAnalysisTx(ctx context.Context, tx *sql.Tx, params CreateStoryAnalysisParams, payloads [6]string) (domain.StoryAnalysis, error) {
	_, err := tx.ExecContext(ctx, sqliteCreateStoryAnalysisSQL,
		params.ID, params.ProjectID, params.EpisodeID,
		sqliteNullable(params.StorySourceID), sqliteNullable(params.WorkflowRunID), sqliteNullable(params.GenerationJobID),
		params.EpisodeID,
		params.Status, params.Summary,
		payloads[0], payloads[1], payloads[2], payloads[3], payloads[4], payloads[5],
	)
	if err != nil {
		if isSQLiteFKViolation(err) {
			return domain.StoryAnalysis{}, domain.ErrNotFound
		}
		return domain.StoryAnalysis{}, err
	}
	return scanSQLiteStoryAnalysis(tx.QueryRowContext(ctx, sqliteGetStoryAnalysisSQL, params.ID))
}

func (r *SQLiteProductionRepository) ListStoryAnalyses(ctx context.Context, episodeID string) ([]domain.StoryAnalysis, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListStoryAnalysesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteStoryAnalyses(rows)
}

func (r *SQLiteProductionRepository) GetStoryAnalysis(ctx context.Context, analysisID string) (domain.StoryAnalysis, error) {
	analysis, err := scanSQLiteStoryAnalysis(r.db.QueryRowContext(ctx, sqliteGetStoryAnalysisSQL, analysisID))
	if err == sql.ErrNoRows {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analysis, err
}

func scanSQLiteStoryAnalyses(rows *sql.Rows) ([]domain.StoryAnalysis, error) {
	analyses := make([]domain.StoryAnalysis, 0)
	for rows.Next() {
		analysis, err := scanSQLiteStoryAnalysis(rows)
		if err != nil {
			return nil, err
		}
		analyses = append(analyses, analysis)
	}
	return analyses, rows.Err()
}

func scanSQLiteStoryAnalysis(row rowScanner) (domain.StoryAnalysis, error) {
	var (
		analysis                          domain.StoryAnalysis
		themes, characters, scenes, props []byte
		outline, agentOutputs             []byte
		createdAt, updatedAt              string
	)
	if err := row.Scan(
		&analysis.ID,
		&analysis.ProjectID,
		&analysis.EpisodeID,
		&analysis.StorySourceID,
		&analysis.WorkflowRunID,
		&analysis.GenerationJobID,
		&analysis.Version,
		&analysis.Status,
		&analysis.Summary,
		&themes,
		&characters,
		&scenes,
		&props,
		&outline,
		&agentOutputs,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.StoryAnalysis{}, err
	}
	if err := decodeStoryAnalysisSeeds(&analysis, themes, characters, scenes, props, outline, agentOutputs); err != nil {
		return domain.StoryAnalysis{}, err
	}
	var err error
	if analysis.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.StoryAnalysis{}, err
	}
	if analysis.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.StoryAnalysis{}, err
	}
	return analysis, nil
}

func scanSQLiteApprovalGates(rows *sql.Rows) ([]domain.ApprovalGate, error) {
	items := make([]domain.ApprovalGate, 0)
	for rows.Next() {
		item, err := scanSQLiteApprovalGate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteApprovalGate(row rowScanner) (domain.ApprovalGate, error) {
	var (
		item                             domain.ApprovalGate
		reviewedAt, createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.WorkflowRunID,
		&item.GateType,
		&item.SubjectType,
		&item.SubjectID,
		&item.Status,
		&item.ReviewedBy,
		&item.ReviewNote,
		&reviewedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.ApprovalGate{}, err
	}
	var err error
	if item.ReviewedAt, err = parseSQLiteTime(reviewedAt); err != nil {
		return domain.ApprovalGate{}, err
	}
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.ApprovalGate{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.ApprovalGate{}, err
	}
	return item, nil
}

func scanSQLiteCharacters(rows *sql.Rows) ([]domain.Character, error) {
	items := make([]domain.Character, 0)
	for rows.Next() {
		item, err := scanSQLiteCharacter(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteCharacter(row rowScanner) (domain.Character, error) {
	var (
		item                 domain.Character
		characterBible       []byte
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.StoryAnalysisID,
		&item.Code,
		&item.Name,
		&item.Description,
		&characterBible,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Character{}, err
	}
	bible, err := decodeCharacterBible(characterBible)
	if err != nil {
		return domain.Character{}, err
	}
	item.CharacterBible = bible
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Character{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Character{}, err
	}
	return item, nil
}

func scanSQLiteScenes(rows *sql.Rows) ([]domain.Scene, error) {
	items := make([]domain.Scene, 0)
	for rows.Next() {
		item, err := scanSQLiteScene(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteScene(row rowScanner) (domain.Scene, error) {
	var (
		item                 domain.Scene
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.StoryAnalysisID,
		&item.Code,
		&item.Name,
		&item.Description,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Scene{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Scene{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Scene{}, err
	}
	return item, nil
}

func scanSQLiteProps(rows *sql.Rows) ([]domain.Prop, error) {
	items := make([]domain.Prop, 0)
	for rows.Next() {
		item, err := scanSQLiteProp(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteProp(row rowScanner) (domain.Prop, error) {
	var (
		item                 domain.Prop
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.StoryAnalysisID,
		&item.Code,
		&item.Name,
		&item.Description,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Prop{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Prop{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Prop{}, err
	}
	return item, nil
}

func scanSQLiteAssets(rows *sql.Rows) ([]domain.Asset, error) {
	items := make([]domain.Asset, 0)
	for rows.Next() {
		item, err := scanSQLiteAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteAsset(row rowScanner) (domain.Asset, error) {
	var (
		item                 domain.Asset
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.Kind,
		&item.Purpose,
		&item.URI,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Asset{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Asset{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Asset{}, err
	}
	return item, nil
}

func scanSQLiteStoryboardShots(rows *sql.Rows) ([]domain.StoryboardShot, error) {
	items := make([]domain.StoryboardShot, 0)
	for rows.Next() {
		item, err := scanSQLiteStoryboardShot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteStoryboardShot(row rowScanner) (domain.StoryboardShot, error) {
	var (
		item                 domain.StoryboardShot
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.EpisodeID,
		&item.StoryAnalysisID,
		&item.SceneID,
		&item.Code,
		&item.Title,
		&item.Description,
		&item.Prompt,
		&item.Position,
		&item.DurationMS,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.StoryboardShot{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.StoryboardShot{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.StoryboardShot{}, err
	}
	return item, nil
}

func scanSQLiteShotPromptPack(row rowScanner) (domain.ShotPromptPack, error) {
	var (
		item                                  domain.ShotPromptPack
		timeSlices, referenceBindings, params []byte
		createdAt, updatedAt                  string
	)
	err := row.Scan(
		&item.ID, &item.ProjectID, &item.EpisodeID, &item.ShotID, &item.Provider,
		&item.Model, &item.Preset, &item.TaskType, &item.DirectPrompt,
		&item.NegativePrompt, &item.IPAdapterStrength, &item.LoRAWeight, &item.LoRACombinationWeight,
		&timeSlices, &referenceBindings, &params,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	if err := json.Unmarshal(timeSlices, &item.TimeSlices); err != nil {
		return domain.ShotPromptPack{}, err
	}
	if err := json.Unmarshal(referenceBindings, &item.ReferenceBindings); err != nil {
		return domain.ShotPromptPack{}, err
	}
	if err := json.Unmarshal(params, &item.Params); err != nil {
		return domain.ShotPromptPack{}, err
	}
	var parseErr error
	if item.CreatedAt, parseErr = parseSQLiteTime(createdAt); parseErr != nil {
		return domain.ShotPromptPack{}, parseErr
	}
	if item.UpdatedAt, parseErr = parseSQLiteTime(updatedAt); parseErr != nil {
		return domain.ShotPromptPack{}, parseErr
	}
	return item, nil
}

func scanSQLiteTimeline(row rowScanner) (domain.Timeline, error) {
	var (
		item                 domain.Timeline
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.EpisodeID,
		&item.Status,
		&item.Version,
		&item.DurationMS,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Timeline{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Timeline{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Timeline{}, err
	}
	return item, nil
}

func scanSQLiteTimelineTracks(rows *sql.Rows) ([]domain.TimelineTrack, error) {
	items := make([]domain.TimelineTrack, 0)
	for rows.Next() {
		item, err := scanSQLiteTimelineTrack(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteTimelineTrack(row rowScanner) (domain.TimelineTrack, error) {
	var (
		item                 domain.TimelineTrack
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.TimelineID,
		&item.Kind,
		&item.Name,
		&item.Position,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.TimelineTrack{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.TimelineTrack{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.TimelineTrack{}, err
	}
	return item, nil
}

func scanSQLiteTimelineClips(rows *sql.Rows) ([]domain.TimelineClip, error) {
	items := make([]domain.TimelineClip, 0)
	for rows.Next() {
		item, err := scanSQLiteTimelineClip(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteTimelineClip(row rowScanner) (domain.TimelineClip, error) {
	var (
		item                 domain.TimelineClip
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.TimelineID,
		&item.TrackID,
		&item.AssetID,
		&item.Kind,
		&item.StartMS,
		&item.DurationMS,
		&item.TrimStartMS,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.TimelineClip{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.TimelineClip{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.TimelineClip{}, err
	}
	return item, nil
}

func scanSQLiteExports(rows *sql.Rows) ([]domain.Export, error) {
	items := make([]domain.Export, 0)
	for rows.Next() {
		item, err := scanSQLiteExport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSQLiteExport(row rowScanner) (domain.Export, error) {
	var (
		item                 domain.Export
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&item.ID,
		&item.TimelineID,
		&item.Status,
		&item.Format,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Export{}, err
	}
	var err error
	if item.CreatedAt, err = parseSQLiteTime(createdAt); err != nil {
		return domain.Export{}, err
	}
	if item.UpdatedAt, err = parseSQLiteTime(updatedAt); err != nil {
		return domain.Export{}, err
	}
	return item, nil
}

func (r *SQLiteProductionRepository) SaveStoryMap(ctx context.Context, params SaveStoryMapParams) (StoryMap, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StoryMap{}, err
	}
	defer tx.Rollback()

	for _, item := range params.Characters {
		if _, err := tx.ExecContext(ctx, sqliteUpsertCharacterSQL,
			item.ID, item.ProjectID, item.EpisodeID, sqliteNullable(item.StoryAnalysisID),
			item.Code, item.Name, item.Description); err != nil {
			return StoryMap{}, sqliteMapFK(err)
		}
	}
	for _, item := range params.Scenes {
		if _, err := tx.ExecContext(ctx, sqliteUpsertSceneSQL,
			item.ID, item.ProjectID, item.EpisodeID, sqliteNullable(item.StoryAnalysisID),
			item.Code, item.Name, item.Description); err != nil {
			return StoryMap{}, sqliteMapFK(err)
		}
	}
	for _, item := range params.Props {
		if _, err := tx.ExecContext(ctx, sqliteUpsertPropSQL,
			item.ID, item.ProjectID, item.EpisodeID, sqliteNullable(item.StoryAnalysisID),
			item.Code, item.Name, item.Description); err != nil {
			return StoryMap{}, sqliteMapFK(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return StoryMap{}, err
	}

	return r.GetStoryMap(ctx, storyMapEpisodeID(params))
}

func (r *SQLiteProductionRepository) GetStoryMap(ctx context.Context, episodeID string) (StoryMap, error) {
	charRows, err := r.db.QueryContext(ctx, sqliteListCharactersSQL, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	defer charRows.Close()
	characters, err := scanSQLiteCharacters(charRows)
	if err != nil {
		return StoryMap{}, err
	}

	sceneRows, err := r.db.QueryContext(ctx, sqliteListScenesSQL, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	defer sceneRows.Close()
	scenes, err := scanSQLiteScenes(sceneRows)
	if err != nil {
		return StoryMap{}, err
	}

	propRows, err := r.db.QueryContext(ctx, sqliteListPropsSQL, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	defer propRows.Close()
	props, err := scanSQLiteProps(propRows)
	if err != nil {
		return StoryMap{}, err
	}

	return StoryMap{Characters: characters, Scenes: scenes, Props: props}, nil
}

func (r *SQLiteProductionRepository) GetCharacter(ctx context.Context, characterID string) (domain.Character, error) {
	character, err := scanSQLiteCharacter(r.db.QueryRowContext(ctx, sqliteGetCharacterSQL, characterID))
	if err == sql.ErrNoRows {
		return domain.Character{}, domain.ErrNotFound
	}
	return character, err
}

func (r *SQLiteProductionRepository) SaveCharacterBible(
	ctx context.Context,
	params SaveCharacterBibleParams,
) (domain.Character, error) {
	payload, err := encodeCharacterBible(params.CharacterBible)
	if err != nil {
		return domain.Character{}, err
	}
	character, err := scanSQLiteCharacter(r.db.QueryRowContext(ctx, sqliteSaveCharacterBibleSQL, payload, params.CharacterID))
	if err == sql.ErrNoRows {
		return domain.Character{}, domain.ErrNotFound
	}
	return character, err
}

func (r *SQLiteProductionRepository) SaveStoryboardShots(ctx context.Context, params SaveStoryboardShotsParams) ([]domain.StoryboardShot, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for _, sp := range params.Shots {
		if _, err := tx.ExecContext(ctx, sqliteUpsertStoryboardShotSQL,
			sp.ID, sp.ProjectID, sp.EpisodeID,
			sqliteNullable(sp.StoryAnalysisID), sqliteNullable(sp.SceneID),
			sp.Code, sp.Title, sp.Description, sp.Prompt, sp.Position, sp.DurationMS,
		); err != nil {
			return nil, sqliteMapFK(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if len(params.Shots) == 0 {
		return []domain.StoryboardShot{}, nil
	}
	return r.ListStoryboardShots(ctx, params.Shots[0].EpisodeID)
}

func (r *SQLiteProductionRepository) ListStoryboardShots(ctx context.Context, episodeID string) ([]domain.StoryboardShot, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListStoryboardShotsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteStoryboardShots(rows)
}

func (r *SQLiteProductionRepository) GetStoryboardShot(ctx context.Context, shotID string) (domain.StoryboardShot, error) {
	shot, err := scanSQLiteStoryboardShot(r.db.QueryRowContext(ctx, sqliteGetStoryboardShotSQL, shotID))
	if err == sql.ErrNoRows {
		return domain.StoryboardShot{}, domain.ErrNotFound
	}
	return shot, err
}

func (r *SQLiteProductionRepository) SaveShotPromptPack(ctx context.Context, params SaveShotPromptPackParams) (domain.ShotPromptPack, error) {
	timeSlices, err := json.Marshal(params.TimeSlices)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	refs, err := json.Marshal(params.ReferenceBindings)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	prms, err := json.Marshal(params.Params)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	_, err = r.db.ExecContext(ctx, sqliteUpsertShotPromptPackSQL,
		params.ID, params.ProjectID, params.EpisodeID, params.ShotID,
		params.Provider, params.Model, params.Preset, params.TaskType,
		params.DirectPrompt, params.NegativePrompt,
		params.IPAdapterStrength, params.LoRAWeight, params.LoRACombinationWeight,
		string(timeSlices), string(refs), string(prms),
	)
	if err != nil {
		return domain.ShotPromptPack{}, sqliteMapFK(err)
	}
	return r.GetShotPromptPack(ctx, params.ShotID)
}

func (r *SQLiteProductionRepository) GetShotPromptPack(ctx context.Context, shotID string) (domain.ShotPromptPack, error) {
	pack, err := scanSQLiteShotPromptPack(r.db.QueryRowContext(ctx, sqliteGetShotPromptPackSQL, shotID))
	if err == sql.ErrNoRows {
		return domain.ShotPromptPack{}, domain.ErrNotFound
	}
	return pack, err
}

func (r *SQLiteProductionRepository) CreateAsset(ctx context.Context, params CreateAssetParams) (domain.Asset, error) {
	asset, err := sqliteCreateAssetTx(ctx, r.db, params)
	return asset, err
}

func (r *SQLiteProductionRepository) ListAssetsByEpisode(ctx context.Context, episodeID string) ([]domain.Asset, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListEpisodeAssetsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteAssets(rows)
}

func (r *SQLiteProductionRepository) GetAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	asset, err := scanSQLiteAsset(r.db.QueryRowContext(ctx, sqliteGetAssetSQL, assetID))
	if err == sql.ErrNoRows {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, err
}

func (r *SQLiteProductionRepository) LockAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	res, err := r.db.ExecContext(ctx, sqliteLockAssetSQL, domain.AssetStatusReady, assetID)
	if err != nil {
		return domain.Asset{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.Asset{}, domain.ErrNotFound
	}
	asset, err := scanSQLiteAsset(r.db.QueryRowContext(ctx, sqliteGetAssetSQL, assetID))
	if err == sql.ErrNoRows {
		return domain.Asset{}, domain.ErrNotFound
	}
	return asset, err
}

func (r *SQLiteProductionRepository) GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error) {
	timeline, err := scanSQLiteTimeline(r.db.QueryRowContext(ctx, sqliteGetEpisodeTimelineSQL, episodeID))
	if err == sql.ErrNoRows {
		return domain.Timeline{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Timeline{}, err
	}
	return r.hydrateTimeline(ctx, timeline)
}

func (r *SQLiteProductionRepository) GetTimelineByID(ctx context.Context, timelineID string) (domain.Timeline, error) {
	timeline, err := scanSQLiteTimeline(r.db.QueryRowContext(ctx, sqliteGetTimelineByIDSQL, timelineID))
	if err == sql.ErrNoRows {
		return domain.Timeline{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Timeline{}, err
	}
	return r.hydrateTimeline(ctx, timeline)
}

func (r *SQLiteProductionRepository) SaveEpisodeTimeline(ctx context.Context, params SaveEpisodeTimelineParams) (domain.Timeline, error) {
	_, err := r.db.ExecContext(ctx, sqliteSaveEpisodeTimelineSQL,
		params.ID, params.EpisodeID, params.Status, params.DurationMS,
	)
	if err != nil {
		return domain.Timeline{}, sqliteMapFK(err)
	}
	return scanSQLiteTimeline(r.db.QueryRowContext(ctx, sqliteGetTimelineByEpisodeForUpdateSQL, params.EpisodeID))
}

func (r *SQLiteProductionRepository) SaveEpisodeTimelineGraph(ctx context.Context, params SaveEpisodeTimelineGraphParams) (domain.Timeline, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Timeline{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, sqliteSaveEpisodeTimelineSQL,
		params.ID, params.EpisodeID, params.Status, params.DurationMS,
	)
	if err != nil {
		return domain.Timeline{}, sqliteMapFK(err)
	}
	timeline, err := scanSQLiteTimeline(tx.QueryRowContext(ctx, sqliteGetTimelineByEpisodeForUpdateSQL, params.EpisodeID))
	if err != nil {
		return domain.Timeline{}, err
	}

	_, err = tx.ExecContext(ctx, sqliteDeleteTimelineTracksSQL, timeline.ID)
	if err != nil {
		return domain.Timeline{}, err
	}

	tracks := make([]domain.TimelineTrack, 0, len(params.Tracks))
	for _, tp := range params.Tracks {
		_, err = tx.ExecContext(ctx, sqliteCreateTimelineTrackSQL,
			tp.ID, timeline.ID, tp.Kind, tp.Name, tp.Position,
		)
		if err != nil {
			return domain.Timeline{}, err
		}
		track, err := scanSQLiteTimelineTrack(tx.QueryRowContext(ctx,
			`SELECT id, timeline_id, kind, name, position, created_at, updated_at FROM timeline_tracks WHERE id = ?`, tp.ID))
		if err != nil {
			return domain.Timeline{}, err
		}
		clips := make([]domain.TimelineClip, 0, len(tp.Clips))
		for _, cp := range tp.Clips {
			_, err = tx.ExecContext(ctx, sqliteCreateTimelineClipSQL,
				cp.ID, timeline.ID, tp.ID, sqliteNullable(cp.AssetID),
				cp.Kind, cp.StartMS, cp.DurationMS, cp.TrimStartMS,
			)
			if err != nil {
				return domain.Timeline{}, sqliteMapFK(err)
			}
			clip, err := scanSQLiteTimelineClip(tx.QueryRowContext(ctx,
				`SELECT id, timeline_id, track_id, COALESCE(asset_id, ''), kind, start_ms, duration_ms, trim_start_ms, created_at, updated_at FROM timeline_clips WHERE id = ?`, cp.ID))
			if err != nil {
				return domain.Timeline{}, err
			}
			clips = append(clips, clip)
		}
		track.Clips = clips
		tracks = append(tracks, track)
	}
	if err := tx.Commit(); err != nil {
		return domain.Timeline{}, err
	}
	timeline.Tracks = tracks
	return timeline, nil
}

func (r *SQLiteProductionRepository) CreateExport(ctx context.Context, params CreateExportParams) (domain.Export, error) {
	_, err := r.db.ExecContext(ctx, sqliteCreateExportSQL,
		params.ID, params.TimelineID, params.Status, params.Format,
	)
	if err != nil {
		return domain.Export{}, sqliteMapFK(err)
	}
	return r.GetExport(ctx, params.ID)
}

func (r *SQLiteProductionRepository) GetExport(ctx context.Context, exportID string) (domain.Export, error) {
	export, err := scanSQLiteExport(r.db.QueryRowContext(ctx, sqliteGetExportSQL, exportID))
	if err == sql.ErrNoRows {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, err
}

func (r *SQLiteProductionRepository) ListExportsByStatus(ctx context.Context, status domain.ExportStatus, limit int) ([]domain.Export, error) {
	rows, err := r.db.QueryContext(ctx, sqliteListExportsByStatusSQL, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLiteExports(rows)
}

func (r *SQLiteProductionRepository) AdvanceExportStatus(ctx context.Context, params AdvanceExportStatusParams) (domain.Export, error) {
	res, err := r.db.ExecContext(ctx, sqliteAdvanceExportStatusSQL, params.To, params.ID, params.From)
	if err != nil {
		return domain.Export{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.Export{}, domain.ErrNotFound
	}
	return r.GetExport(ctx, params.ID)
}

// helpers

func (r *SQLiteProductionRepository) getStorySourceByID(ctx context.Context, id string) (domain.StorySource, error) {
	return scanStorySource(r.db.QueryRowContext(ctx, sqliteGetStorySourceByIDSQL, id))
}

func (r *SQLiteProductionRepository) hydrateTimeline(ctx context.Context, timeline domain.Timeline) (domain.Timeline, error) {
	trackRows, err := r.db.QueryContext(ctx, sqliteListTimelineTracksSQL, timeline.ID)
	if err != nil {
		return domain.Timeline{}, err
	}
	defer trackRows.Close()
	tracks, err := scanSQLiteTimelineTracks(trackRows)
	if err != nil {
		return domain.Timeline{}, err
	}

	clipRows, err := r.db.QueryContext(ctx, sqliteListTimelineClipsSQL, timeline.ID)
	if err != nil {
		return domain.Timeline{}, err
	}
	defer clipRows.Close()
	clips, err := scanSQLiteTimelineClips(clipRows)
	if err != nil {
		return domain.Timeline{}, err
	}

	timeline.Tracks = attachTimelineClips(tracks, clips)
	return timeline, nil
}

type sqliteExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func sqliteCreateAssetTx(ctx context.Context, db sqliteExecer, params CreateAssetParams) (domain.Asset, error) {
	existing, err := scanSQLiteAsset(db.QueryRowContext(ctx, sqliteGetExistingAssetSQL,
		params.EpisodeID, params.Kind, params.Purpose, params.URI,
	))
	if err == nil {
		return existing, nil
	}
	_, err = db.ExecContext(ctx, sqliteCreateAssetSQL,
		params.ID, params.ProjectID, params.EpisodeID, params.Kind, params.Purpose, params.URI, params.Status,
		// the WHERE NOT EXISTS clause reuses positional params ?3-?6
	)
	if err != nil {
		return domain.Asset{}, sqliteMapFK(err)
	}
	asset, err := scanSQLiteAsset(db.QueryRowContext(ctx, sqliteGetAssetSQL, params.ID))
	if err == sql.ErrNoRows {
		return scanSQLiteAsset(db.QueryRowContext(ctx, sqliteGetExistingAssetSQL,
			params.EpisodeID, params.Kind, params.Purpose, params.URI,
		))
	}
	return asset, err
}

func sqliteNullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func sqliteMapFK(err error) error {
	if isSQLiteFKViolation(err) {
		return domain.ErrNotFound
	}
	return err
}

func storyMapEpisodeID(params SaveStoryMapParams) string {
	if len(params.Characters) > 0 {
		return params.Characters[0].EpisodeID
	}
	if len(params.Scenes) > 0 {
		return params.Scenes[0].EpisodeID
	}
	if len(params.Props) > 0 {
		return params.Props[0].EpisodeID
	}
	return ""
}

// Batch generation feature stub implementations for SQLite
func (r *SQLiteProductionRepository) RetryJob(_ context.Context, _ string) (domain.GenerationJob, error) {
	return domain.GenerationJob{}, nil
}

func (r *SQLiteProductionRepository) UpdateJobPriority(_ context.Context, _ string, _ int) error {
	return nil
}

func (r *SQLiteProductionRepository) ListJobsByPriority(_ context.Context, _ string, _ domain.GenerationJobStatus) ([]domain.GenerationJob, error) {
	return []domain.GenerationJob{}, nil
}

func (r *SQLiteProductionRepository) PauseEpisodeQueue(_ context.Context, _ string) error {
	return nil
}

func (r *SQLiteProductionRepository) ResumeEpisodeQueue(_ context.Context, _ string) error {
	return nil
}

func (r *SQLiteProductionRepository) IsEpisodeQueuePaused(_ context.Context, _ string) (bool, error) {
	return false, nil
}
