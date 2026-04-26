package domain

import "time"

type Project struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	Status         ProjectStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Episode struct {
	ID        string
	ProjectID string
	Number    int
	Title     string
	Status    EpisodeStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
