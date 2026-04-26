package domain

import "time"

type WorkflowRun struct {
	ID        string
	ProjectID string
	EpisodeID string
	Status    WorkflowRunStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GenerationJob struct {
	ID            string
	ProjectID     string
	EpisodeID     string
	WorkflowRunID string
	Provider      string
	Model         string
	TaskType      string
	Status        GenerationJobStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Timeline struct {
	ID         string
	EpisodeID  string
	Status     TimelineStatus
	Version    int
	DurationMS int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
