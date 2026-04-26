package httpapi

import (
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type projectResponse struct {
	ID             string               `json:"id"`
	OrganizationID string               `json:"organization_id"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Status         domain.ProjectStatus `json:"status"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type episodeResponse struct {
	ID        string               `json:"id"`
	ProjectID string               `json:"project_id"`
	Number    int                  `json:"number"`
	Title     string               `json:"title"`
	Status    domain.EpisodeStatus `json:"status"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

func projectDTO(project domain.Project) projectResponse {
	return projectResponse{
		ID:             project.ID,
		OrganizationID: project.OrganizationID,
		Name:           project.Name,
		Description:    project.Description,
		Status:         project.Status,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}
}

func episodeDTO(episode domain.Episode) episodeResponse {
	return episodeResponse{
		ID:        episode.ID,
		ProjectID: episode.ProjectID,
		Number:    episode.Number,
		Title:     episode.Title,
		Status:    episode.Status,
		CreatedAt: episode.CreatedAt,
		UpdatedAt: episode.UpdatedAt,
	}
}
