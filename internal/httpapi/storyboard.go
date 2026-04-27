package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

func (api *api) listStoryboardShots(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	shots, err := api.productionService.ListStoryboardShots(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"storyboard_shots": storyboardShotDTOs(shots)})
}

func (api *api) seedStoryboardShots(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	shots, err := api.productionService.SeedStoryboardShots(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"storyboard_shots": storyboardShotDTOs(shots)})
}

type updateStoryboardShotRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	DurationMS  int    `json:"duration_ms"`
}

func (api *api) updateStoryboardShot(w http.ResponseWriter, r *http.Request) {
	var request updateStoryboardShotRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	shot, err := api.productionService.UpdateStoryboardShot(
		r.Context(),
		chi.URLParam(r, "shotId"),
		service.UpdateStoryboardShotInput{
			Title:       request.Title,
			Description: request.Description,
			Prompt:      request.Prompt,
			DurationMS:  request.DurationMS,
		},
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"storyboard_shot": storyboardShotDTOs([]domain.StoryboardShot{shot})[0]})
}
