package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createEpisodeRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

func (api *api) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := api.projectService.ListProjects(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectDTO(project))
	}
	writeJSON(w, http.StatusOK, Envelope{"projects": items})
}

func (api *api) createProject(w http.ResponseWriter, r *http.Request) {
	var request createProjectRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}

	project, err := api.projectService.CreateProject(r.Context(), service.CreateProjectInput{
		Name:        request.Name,
		Description: request.Description,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"project": projectDTO(project)})
}

func (api *api) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := api.projectService.GetProject(r.Context(), chi.URLParam(r, "projectId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"project": projectDTO(project)})
}

func (api *api) listEpisodes(w http.ResponseWriter, r *http.Request) {
	episodes, err := api.projectService.ListEpisodes(r.Context(), chi.URLParam(r, "projectId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]episodeResponse, 0, len(episodes))
	for _, episode := range episodes {
		items = append(items, episodeDTO(episode))
	}
	writeJSON(w, http.StatusOK, Envelope{"episodes": items})
}

func (api *api) createEpisode(w http.ResponseWriter, r *http.Request) {
	var request createEpisodeRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}

	episode, err := api.projectService.CreateEpisode(r.Context(), service.CreateEpisodeInput{
		ProjectID: chi.URLParam(r, "projectId"),
		Number:    request.Number,
		Title:     request.Title,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"episode": episodeDTO(episode)})
}

func (api *api) getEpisode(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"episode": episodeDTO(episode)})
}
