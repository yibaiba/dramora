package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/repo"
)

func (s *ProductionService) GetEpisodeTimeline(ctx context.Context, episodeID string) (domain.Timeline, error) {
	if strings.TrimSpace(episodeID) == "" {
		return domain.Timeline{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	if err := s.authorizeEpisode(ctx, episodeID); err != nil {
		return domain.Timeline{}, err
	}
	return s.production.GetEpisodeTimeline(ctx, episodeID)
}

func (s *ProductionService) SaveEpisodeTimeline(
	ctx context.Context,
	input SaveTimelineInput,
) (domain.Timeline, error) {
	if strings.TrimSpace(input.EpisodeID) == "" {
		return domain.Timeline{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	if input.DurationMS < 0 {
		return domain.Timeline{}, fmt.Errorf("%w: duration_ms must be non-negative", domain.ErrInvalidInput)
	}
	if err := validateTimelineInput(input); err != nil {
		return domain.Timeline{}, err
	}

	id, err := domain.NewID()
	if err != nil {
		return domain.Timeline{}, err
	}

	if len(input.Tracks) == 0 {
		return s.production.SaveEpisodeTimeline(ctx, repo.SaveEpisodeTimelineParams{
			ID:         id,
			EpisodeID:  input.EpisodeID,
			Status:     domain.TimelineStatusSaved,
			DurationMS: input.DurationMS,
		})
	}

	tracks, err := timelineTrackParams(input.Tracks)
	if err != nil {
		return domain.Timeline{}, err
	}
	return s.production.SaveEpisodeTimelineGraph(ctx, repo.SaveEpisodeTimelineGraphParams{
		ID:         id,
		EpisodeID:  input.EpisodeID,
		Status:     domain.TimelineStatusSaved,
		DurationMS: input.DurationMS,
		Tracks:     tracks,
	})
}

func (s *ProductionService) StartEpisodeExport(ctx context.Context, episodeID string) (domain.Export, error) {
	timeline, err := s.GetEpisodeTimeline(ctx, episodeID)
	if err != nil {
		return domain.Export{}, err
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.Export{}, err
	}
	export, err := s.production.CreateExport(ctx, repo.CreateExportParams{
		ID:         id,
		TimelineID: timeline.ID,
		Status:     domain.ExportStatusQueued,
		Format:     "mp4",
	})
	if err != nil {
		return domain.Export{}, err
	}
	if err := s.jobClient.Enqueue(ctx, jobs.Job{
		ID:   id,
		Kind: jobs.JobKindExportRender,
		Payload: map[string]any{
			"export_id":   id,
			"timeline_id": timeline.ID,
		},
	}); err != nil {
		return domain.Export{}, err
	}
	return export, nil
}

func (s *ProductionService) GetExport(ctx context.Context, id string) (domain.Export, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Export{}, fmt.Errorf("%w: export id is required", domain.ErrInvalidInput)
	}
	export, err := s.production.GetExport(ctx, id)
	if err != nil {
		return domain.Export{}, err
	}
	timeline, err := s.production.GetTimelineByID(ctx, export.TimelineID)
	if err != nil {
		return domain.Export{}, err
	}
	if err := s.authorizeEpisode(ctx, timeline.EpisodeID); err != nil {
		return domain.Export{}, err
	}
	return export, nil
}

func timelineTrackParams(
	inputs []SaveTimelineTrackInput,
) ([]repo.SaveTimelineTrackParams, error) {
	tracks := make([]repo.SaveTimelineTrackParams, 0, len(inputs))
	for _, input := range inputs {
		id, err := domain.NewID()
		if err != nil {
			return nil, err
		}
		clips, err := timelineClipParams(input.Clips)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, repo.SaveTimelineTrackParams{
			ID: id, Kind: input.Kind, Name: input.Name, Position: input.Position, Clips: clips,
		})
	}
	return tracks, nil
}

func timelineClipParams(inputs []SaveTimelineClipInput) ([]repo.SaveTimelineClipParams, error) {
	clips := make([]repo.SaveTimelineClipParams, 0, len(inputs))
	for _, input := range inputs {
		id, err := domain.NewID()
		if err != nil {
			return nil, err
		}
		clips = append(clips, repo.SaveTimelineClipParams{
			ID: id, AssetID: input.AssetID, Kind: input.Kind, StartMS: input.StartMS,
			DurationMS: input.DurationMS, TrimStartMS: input.TrimStartMS,
		})
	}
	return clips, nil
}

func validateTimelineInput(input SaveTimelineInput) error {
	for _, track := range input.Tracks {
		if err := validateTimelineTrack(track, input.DurationMS); err != nil {
			return err
		}
	}
	return nil
}

func validateTimelineTrack(track SaveTimelineTrackInput, durationMS int) error {
	if strings.TrimSpace(track.Kind) == "" {
		return fmt.Errorf("%w: track kind is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(track.Name) == "" {
		return fmt.Errorf("%w: track name is required", domain.ErrInvalidInput)
	}
	for _, clip := range track.Clips {
		if err := validateTimelineClip(clip, durationMS); err != nil {
			return err
		}
	}
	return nil
}

func validateTimelineClip(clip SaveTimelineClipInput, durationMS int) error {
	if strings.TrimSpace(clip.Kind) == "" {
		return fmt.Errorf("%w: clip kind is required", domain.ErrInvalidInput)
	}
	if clip.StartMS < 0 || clip.DurationMS < 0 || clip.TrimStartMS < 0 {
		return fmt.Errorf("%w: clip timing must be non-negative", domain.ErrInvalidInput)
	}
	if durationMS > 0 && clip.StartMS+clip.DurationMS > durationMS {
		return fmt.Errorf("%w: clip exceeds timeline duration", domain.ErrInvalidInput)
	}
	return nil
}
