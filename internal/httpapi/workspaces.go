package httpapi

import (
	"net/http"

	"github.com/yibaiba/dramora/internal/service"
)

type workspaceResponse struct {
	ID              string `json:"id"`
	OrganizationID  string `json:"organization_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	MembersCount    int    `json:"members_count"`
	ProjectsCount   int    `json:"projects_count"`
	CreatedAt       string `json:"created_at"`
}

func workspaceDTO(ws service.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:             ws.ID,
		OrganizationID: ws.OrganizationID,
		Name:           ws.Name,
		Description:    ws.Description,
		MembersCount:   ws.MembersCount,
		ProjectsCount:  ws.ProjectsCount,
		CreatedAt:      ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (api *api) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := api.projectService.ListWorkspaces(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]workspaceResponse, 0, len(workspaces))
	for _, ws := range workspaces {
		items = append(items, workspaceDTO(ws))
	}
	writeJSON(w, http.StatusOK, Envelope{"workspaces": items})
}
