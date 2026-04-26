package repo

import "github.com/yibaiba/dramora/internal/domain"

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
