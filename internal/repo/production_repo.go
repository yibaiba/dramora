package repo

import (
	"context"
	"encoding/json"
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
	CompleteStoryAnalysisJob(ctx context.Context, params CompleteStoryAnalysisJobParams) (StoryAnalysisCompletion, error)
	CreateStoryAnalysis(ctx context.Context, params CreateStoryAnalysisParams) (domain.StoryAnalysis, error)
	ListStoryAnalyses(ctx context.Context, episodeID string) ([]domain.StoryAnalysis, error)
	GetStoryAnalysis(ctx context.Context, analysisID string) (domain.StoryAnalysis, error)
	SaveStoryMap(ctx context.Context, params SaveStoryMapParams) (StoryMap, error)
	GetStoryMap(ctx context.Context, episodeID string) (StoryMap, error)
	SaveStoryboardShots(ctx context.Context, params SaveStoryboardShotsParams) ([]domain.StoryboardShot, error)
	ListStoryboardShots(ctx context.Context, episodeID string) ([]domain.StoryboardShot, error)
	GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error)
	SaveEpisodeTimeline(ctx context.Context, params SaveEpisodeTimelineParams) (domain.Timeline, error)
	SaveEpisodeTimelineGraph(ctx context.Context, params SaveEpisodeTimelineGraphParams) (domain.Timeline, error)
	CreateExport(ctx context.Context, params CreateExportParams) (domain.Export, error)
	GetExport(ctx context.Context, exportID string) (domain.Export, error)
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

type PostgresProductionRepository struct {
	pool *pgxpool.Pool
}

type storyAnalysisQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

func advanceGenerationJobStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdvanceGenerationJobStatusParams,
) (domain.GenerationJob, error) {
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
	return job, nil
}

func createStoryAnalysisWithPayloads(
	ctx context.Context,
	queryer storyAnalysisQueryer,
	params CreateStoryAnalysisParams,
	payloads [4]string,
) (domain.StoryAnalysis, error) {
	analysis, err := scanStoryAnalysis(queryer.QueryRow(ctx, createStoryAnalysisSQL,
		params.ID,
		params.ProjectID,
		params.EpisodeID,
		nullableUUID(params.WorkflowRunID),
		nullableUUID(params.GenerationJobID),
		params.Status,
		params.Summary,
		payloads[0],
		payloads[1],
		payloads[2],
		payloads[3],
	))
	if isForeignKeyViolation(err) {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analysis, err
}

func storyAnalysisPayloads(params CreateStoryAnalysisParams) ([4]string, error) {
	values := [][]string{
		params.Themes,
		params.CharacterSeeds,
		params.SceneSeeds,
		params.PropSeeds,
	}
	var payloads [4]string
	for index, value := range values {
		payload, err := json.Marshal(value)
		if err != nil {
			return [4]string{}, err
		}
		payloads[index] = string(payload)
	}
	return payloads, nil
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
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

func (r *PostgresProductionRepository) SaveStoryMap(
	ctx context.Context,
	params SaveStoryMapParams,
) (StoryMap, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoryMap{}, err
	}
	defer tx.Rollback(ctx)

	storyMap, err := saveStoryMapTx(ctx, tx, params)
	if err != nil {
		return StoryMap{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoryMap{}, err
	}
	return storyMap, nil
}

func (r *PostgresProductionRepository) GetStoryMap(ctx context.Context, episodeID string) (StoryMap, error) {
	characters, err := r.listCharacters(ctx, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	scenes, err := r.listScenes(ctx, episodeID)
	if err != nil {
		return StoryMap{}, err
	}
	props, err := r.listProps(ctx, episodeID)
	return StoryMap{Characters: characters, Scenes: scenes, Props: props}, err
}

func (r *PostgresProductionRepository) SaveStoryboardShots(
	ctx context.Context,
	params SaveStoryboardShotsParams,
) ([]domain.StoryboardShot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	shots := make([]domain.StoryboardShot, 0, len(params.Shots))
	for _, shotParams := range params.Shots {
		shot, err := scanStoryboardShot(tx.QueryRow(ctx, upsertStoryboardShotSQL,
			shotParams.ID, shotParams.ProjectID, shotParams.EpisodeID,
			nullableUUID(shotParams.StoryAnalysisID), nullableUUID(shotParams.SceneID),
			shotParams.Code, shotParams.Title, shotParams.Description, shotParams.Prompt,
			shotParams.Position, shotParams.DurationMS,
		))
		if err != nil {
			return nil, mapForeignKeyViolation(err)
		}
		shots = append(shots, shot)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return shots, nil
}

func (r *PostgresProductionRepository) ListStoryboardShots(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryboardShot, error) {
	rows, err := r.pool.Query(ctx, listStoryboardShotsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStoryboardShots(rows)
}

func (r *PostgresProductionRepository) GetEpisodeTimeline(
	ctx context.Context,
	episodeID string,
) (domain.Timeline, error) {
	timeline, err := scanTimeline(r.pool.QueryRow(ctx, getEpisodeTimelineSQL, episodeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Timeline{}, domain.ErrNotFound
	}
	if err != nil {
		return timeline, err
	}
	return r.hydrateTimeline(ctx, timeline)
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

func (r *PostgresProductionRepository) SaveEpisodeTimelineGraph(
	ctx context.Context,
	params SaveEpisodeTimelineGraphParams,
) (domain.Timeline, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Timeline{}, err
	}
	defer tx.Rollback(ctx)

	timeline, err := scanTimeline(tx.QueryRow(ctx, saveEpisodeTimelineSQL,
		params.ID, params.EpisodeID, params.Status, params.DurationMS,
	))
	if err != nil {
		return domain.Timeline{}, mapForeignKeyViolation(err)
	}
	if _, err := tx.Exec(ctx, deleteTimelineTracksSQL, timeline.ID); err != nil {
		return domain.Timeline{}, err
	}
	tracks, err := saveTimelineTracksTx(ctx, tx, timeline.ID, params.Tracks)
	if err != nil {
		return domain.Timeline{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Timeline{}, err
	}
	timeline.Tracks = tracks
	return timeline, nil
}

func (r *PostgresProductionRepository) CreateExport(
	ctx context.Context,
	params CreateExportParams,
) (domain.Export, error) {
	export, err := scanExport(r.pool.QueryRow(ctx, createExportSQL,
		params.ID, params.TimelineID, params.Status, params.Format,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) GetExport(ctx context.Context, exportID string) (domain.Export, error) {
	export, err := scanExport(r.pool.QueryRow(ctx, getExportSQL, exportID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, err
}

func saveStoryMapTx(ctx context.Context, tx pgx.Tx, params SaveStoryMapParams) (StoryMap, error) {
	storyMap := StoryMap{
		Characters: make([]domain.Character, 0, len(params.Characters)),
		Scenes:     make([]domain.Scene, 0, len(params.Scenes)),
		Props:      make([]domain.Prop, 0, len(params.Props)),
	}
	for _, item := range params.Characters {
		value, err := scanCharacter(tx.QueryRow(ctx, upsertCharacterSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Characters = append(storyMap.Characters, value)
	}
	for _, item := range params.Scenes {
		value, err := scanScene(tx.QueryRow(ctx, upsertSceneSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Scenes = append(storyMap.Scenes, value)
	}
	for _, item := range params.Props {
		value, err := scanProp(tx.QueryRow(ctx, upsertPropSQL,
			item.ID, item.ProjectID, item.EpisodeID, nullableUUID(item.StoryAnalysisID),
			item.Code, item.Name, item.Description))
		if err != nil {
			return StoryMap{}, mapForeignKeyViolation(err)
		}
		storyMap.Props = append(storyMap.Props, value)
	}
	return storyMap, nil
}

func saveTimelineTracksTx(
	ctx context.Context,
	tx pgx.Tx,
	timelineID string,
	params []SaveTimelineTrackParams,
) ([]domain.TimelineTrack, error) {
	tracks := make([]domain.TimelineTrack, 0, len(params))
	for _, trackParams := range params {
		track, err := scanTimelineTrack(tx.QueryRow(ctx, createTimelineTrackSQL,
			trackParams.ID, timelineID, trackParams.Kind, trackParams.Name, trackParams.Position,
		))
		if err != nil {
			return nil, err
		}
		track.Clips, err = saveTimelineClipsTx(ctx, tx, timelineID, track.ID, trackParams.Clips)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func saveTimelineClipsTx(
	ctx context.Context,
	tx pgx.Tx,
	timelineID string,
	trackID string,
	params []SaveTimelineClipParams,
) ([]domain.TimelineClip, error) {
	clips := make([]domain.TimelineClip, 0, len(params))
	for _, clipParams := range params {
		clip, err := scanTimelineClip(tx.QueryRow(ctx, createTimelineClipSQL,
			clipParams.ID, timelineID, trackID, nullableUUID(clipParams.AssetID),
			clipParams.Kind, clipParams.StartMS, clipParams.DurationMS, clipParams.TrimStartMS,
		))
		if err != nil {
			return nil, mapForeignKeyViolation(err)
		}
		clips = append(clips, clip)
	}
	return clips, nil
}

func (r *PostgresProductionRepository) hydrateTimeline(
	ctx context.Context,
	timeline domain.Timeline,
) (domain.Timeline, error) {
	tracks, err := r.listTimelineTracks(ctx, timeline.ID)
	if err != nil {
		return domain.Timeline{}, err
	}
	clips, err := r.listTimelineClips(ctx, timeline.ID)
	if err != nil {
		return domain.Timeline{}, err
	}
	timeline.Tracks = attachTimelineClips(tracks, clips)
	return timeline, nil
}

func attachTimelineClips(
	tracks []domain.TimelineTrack,
	clips []domain.TimelineClip,
) []domain.TimelineTrack {
	trackByID := make(map[string]int, len(tracks))
	for index := range tracks {
		tracks[index].Clips = []domain.TimelineClip{}
		trackByID[tracks[index].ID] = index
	}
	for _, clip := range clips {
		index, ok := trackByID[clip.TrackID]
		if ok {
			tracks[index].Clips = append(tracks[index].Clips, clip)
		}
	}
	return tracks
}

func (r *PostgresProductionRepository) listCharacters(ctx context.Context, episodeID string) ([]domain.Character, error) {
	rows, err := r.pool.Query(ctx, listCharactersSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCharacters(rows)
}

func (r *PostgresProductionRepository) listScenes(ctx context.Context, episodeID string) ([]domain.Scene, error) {
	rows, err := r.pool.Query(ctx, listScenesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScenes(rows)
}

func (r *PostgresProductionRepository) listProps(ctx context.Context, episodeID string) ([]domain.Prop, error) {
	rows, err := r.pool.Query(ctx, listPropsSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProps(rows)
}

func (r *PostgresProductionRepository) listTimelineTracks(ctx context.Context, timelineID string) ([]domain.TimelineTrack, error) {
	rows, err := r.pool.Query(ctx, listTimelineTracksSQL, timelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTimelineTracks(rows)
}

func (r *PostgresProductionRepository) listTimelineClips(ctx context.Context, timelineID string) ([]domain.TimelineClip, error) {
	rows, err := r.pool.Query(ctx, listTimelineClipsSQL, timelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTimelineClips(rows)
}

func mapForeignKeyViolation(err error) error {
	if isForeignKeyViolation(err) {
		return domain.ErrNotFound
	}
	return err
}
