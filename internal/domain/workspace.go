package domain

import "time"

type Workspace struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	MembersCount   int
	ProjectsCount  int
	CreatedAt      time.Time
}
