package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

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

func (r *PostgresProductionRepository) GetTimelineByID(
	ctx context.Context,
	timelineID string,
) (domain.Timeline, error) {
	timeline, err := scanTimeline(r.pool.QueryRow(ctx, getTimelineByIDSQL, timelineID))
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
