package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (api *api) seedEpisodeProduction(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result, err := api.productionService.SeedEpisodeProduction(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{
		"approval_gates":   approvalGateDTOs(result.ApprovalGates),
		"assets":           assetDTOs(result.Assets),
		"story_map":        storyMapDTO(result.StoryMap),
		"storyboard_shots": storyboardShotDTOs(result.StoryboardShots),
	})
}
