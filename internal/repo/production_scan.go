package repo

import (
	"encoding/json"

	"github.com/yibaiba/dramora/internal/domain"
)

func scanWorkflowRun(row rowScanner) (domain.WorkflowRun, error) {
	var run domain.WorkflowRun
	if err := row.Scan(
		&run.ID,
		&run.ProjectID,
		&run.EpisodeID,
		&run.Status,
		&run.CreatedAt,
		&run.UpdatedAt,
	); err != nil {
		return domain.WorkflowRun{}, err
	}
	return run, nil
}

func scanGenerationJobs(rows rowsScanner) ([]domain.GenerationJob, error) {
	jobs := make([]domain.GenerationJob, 0)
	for rows.Next() {
		job, err := scanGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func scanGenerationJob(row rowScanner) (domain.GenerationJob, error) {
	var job domain.GenerationJob
	if err := row.Scan(
		&job.ID,
		&job.ProjectID,
		&job.EpisodeID,
		&job.WorkflowRunID,
		&job.Provider,
		&job.Model,
		&job.TaskType,
		&job.Status,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func scanStoryAnalyses(rows rowsScanner) ([]domain.StoryAnalysis, error) {
	analyses := make([]domain.StoryAnalysis, 0)
	for rows.Next() {
		analysis, err := scanStoryAnalysis(rows)
		if err != nil {
			return nil, err
		}
		analyses = append(analyses, analysis)
	}
	return analyses, rows.Err()
}

func scanStoryAnalysis(row rowScanner) (domain.StoryAnalysis, error) {
	var analysis domain.StoryAnalysis
	var themes, characters, scenes, props []byte
	if err := row.Scan(
		&analysis.ID,
		&analysis.ProjectID,
		&analysis.EpisodeID,
		&analysis.WorkflowRunID,
		&analysis.GenerationJobID,
		&analysis.Version,
		&analysis.Status,
		&analysis.Summary,
		&themes,
		&characters,
		&scenes,
		&props,
		&analysis.CreatedAt,
		&analysis.UpdatedAt,
	); err != nil {
		return domain.StoryAnalysis{}, err
	}
	if err := decodeStoryAnalysisSeeds(&analysis, themes, characters, scenes, props); err != nil {
		return domain.StoryAnalysis{}, err
	}
	return analysis, nil
}

func decodeStoryAnalysisSeeds(
	analysis *domain.StoryAnalysis,
	themes []byte,
	characters []byte,
	scenes []byte,
	props []byte,
) error {
	if err := json.Unmarshal(themes, &analysis.Themes); err != nil {
		return err
	}
	if err := json.Unmarshal(characters, &analysis.CharacterSeeds); err != nil {
		return err
	}
	if err := json.Unmarshal(scenes, &analysis.SceneSeeds); err != nil {
		return err
	}
	return json.Unmarshal(props, &analysis.PropSeeds)
}

func scanTimeline(row rowScanner) (domain.Timeline, error) {
	var timeline domain.Timeline
	if err := row.Scan(
		&timeline.ID,
		&timeline.EpisodeID,
		&timeline.Status,
		&timeline.Version,
		&timeline.DurationMS,
		&timeline.CreatedAt,
		&timeline.UpdatedAt,
	); err != nil {
		return domain.Timeline{}, err
	}
	return timeline, nil
}

func scanCharacters(rows rowsScanner) ([]domain.Character, error) {
	items := make([]domain.Character, 0)
	for rows.Next() {
		item, err := scanCharacter(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanCharacter(row rowScanner) (domain.Character, error) {
	var item domain.Character
	err := row.Scan(&item.ID, &item.ProjectID, &item.EpisodeID, &item.StoryAnalysisID,
		&item.Code, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanScenes(rows rowsScanner) ([]domain.Scene, error) {
	items := make([]domain.Scene, 0)
	for rows.Next() {
		item, err := scanScene(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanScene(row rowScanner) (domain.Scene, error) {
	var item domain.Scene
	err := row.Scan(&item.ID, &item.ProjectID, &item.EpisodeID, &item.StoryAnalysisID,
		&item.Code, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanProps(rows rowsScanner) ([]domain.Prop, error) {
	items := make([]domain.Prop, 0)
	for rows.Next() {
		item, err := scanProp(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanProp(row rowScanner) (domain.Prop, error) {
	var item domain.Prop
	err := row.Scan(&item.ID, &item.ProjectID, &item.EpisodeID, &item.StoryAnalysisID,
		&item.Code, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanAssets(rows rowsScanner) ([]domain.Asset, error) {
	items := make([]domain.Asset, 0)
	for rows.Next() {
		item, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAsset(row rowScanner) (domain.Asset, error) {
	var item domain.Asset
	err := row.Scan(&item.ID, &item.ProjectID, &item.EpisodeID, &item.Kind,
		&item.Purpose, &item.URI, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanStoryboardShots(rows rowsScanner) ([]domain.StoryboardShot, error) {
	items := make([]domain.StoryboardShot, 0)
	for rows.Next() {
		item, err := scanStoryboardShot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanStoryboardShot(row rowScanner) (domain.StoryboardShot, error) {
	var item domain.StoryboardShot
	err := row.Scan(&item.ID, &item.ProjectID, &item.EpisodeID, &item.StoryAnalysisID,
		&item.SceneID, &item.Code, &item.Title, &item.Description, &item.Prompt,
		&item.Position, &item.DurationMS, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanShotPromptPack(row rowScanner) (domain.ShotPromptPack, error) {
	var item domain.ShotPromptPack
	var timeSlices, referenceBindings, params []byte
	err := row.Scan(
		&item.ID, &item.ProjectID, &item.EpisodeID, &item.ShotID, &item.Provider,
		&item.Model, &item.Preset, &item.TaskType, &item.DirectPrompt,
		&item.NegativePrompt, &timeSlices, &referenceBindings, &params,
		&item.CreatedAt, &item.UpdatedAt,
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
	return item, nil
}

func scanTimelineTracks(rows rowsScanner) ([]domain.TimelineTrack, error) {
	items := make([]domain.TimelineTrack, 0)
	for rows.Next() {
		item, err := scanTimelineTrack(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanTimelineTrack(row rowScanner) (domain.TimelineTrack, error) {
	var item domain.TimelineTrack
	err := row.Scan(&item.ID, &item.TimelineID, &item.Kind, &item.Name,
		&item.Position, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanTimelineClips(rows rowsScanner) ([]domain.TimelineClip, error) {
	items := make([]domain.TimelineClip, 0)
	for rows.Next() {
		item, err := scanTimelineClip(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanTimelineClip(row rowScanner) (domain.TimelineClip, error) {
	var item domain.TimelineClip
	err := row.Scan(&item.ID, &item.TimelineID, &item.TrackID, &item.AssetID, &item.Kind,
		&item.StartMS, &item.DurationMS, &item.TrimStartMS, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanExport(row rowScanner) (domain.Export, error) {
	var item domain.Export
	err := row.Scan(&item.ID, &item.TimelineID, &item.Status, &item.Format, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
