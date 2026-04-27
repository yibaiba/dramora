package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (api *api) getStoryMap(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	storyMap, err := api.productionService.GetStoryMap(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"story_map": storyMapDTO(storyMap)})
}

func (api *api) seedStoryMap(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	storyMap, err := api.productionService.SeedStoryMap(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"story_map": storyMapDTO(storyMap)})
}
