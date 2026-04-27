package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
