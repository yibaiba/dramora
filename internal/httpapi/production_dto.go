package httpapi

import (
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type workflowRunResponse struct {
	ID        string                   `json:"id"`
	ProjectID string                   `json:"project_id"`
	EpisodeID string                   `json:"episode_id"`
	Status    domain.WorkflowRunStatus `json:"status"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type generationJobResponse struct {
	ID            string                     `json:"id"`
	ProjectID     string                     `json:"project_id"`
	EpisodeID     string                     `json:"episode_id"`
	WorkflowRunID string                     `json:"workflow_run_id"`
	Provider      string                     `json:"provider"`
	Model         string                     `json:"model"`
	TaskType      string                     `json:"task_type"`
	Status        domain.GenerationJobStatus `json:"status"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type timelineResponse struct {
	ID         string                `json:"id"`
	EpisodeID  string                `json:"episode_id"`
	Status     domain.TimelineStatus `json:"status"`
	Version    int                   `json:"version"`
	DurationMS int                   `json:"duration_ms"`
	Tracks     []Envelope            `json:"tracks"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

func workflowRunDTO(run domain.WorkflowRun) workflowRunResponse {
	return workflowRunResponse{
		ID:        run.ID,
		ProjectID: run.ProjectID,
		EpisodeID: run.EpisodeID,
		Status:    run.Status,
		CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt,
	}
}

func generationJobDTO(job domain.GenerationJob) generationJobResponse {
	return generationJobResponse{
		ID:            job.ID,
		ProjectID:     job.ProjectID,
		EpisodeID:     job.EpisodeID,
		WorkflowRunID: job.WorkflowRunID,
		Provider:      job.Provider,
		Model:         job.Model,
		TaskType:      job.TaskType,
		Status:        job.Status,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
	}
}

func timelineDTO(timeline domain.Timeline) timelineResponse {
	return timelineResponse{
		ID:         timeline.ID,
		EpisodeID:  timeline.EpisodeID,
		Status:     timeline.Status,
		Version:    timeline.Version,
		DurationMS: timeline.DurationMS,
		Tracks:     []Envelope{},
		CreatedAt:  timeline.CreatedAt,
		UpdatedAt:  timeline.UpdatedAt,
	}
}
