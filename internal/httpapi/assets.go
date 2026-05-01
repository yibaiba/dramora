package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (api *api) listEpisodeAssets(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	assets, err := api.productionService.ListEpisodeAssets(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"assets": assetDTOs(assets)})
}

func (api *api) seedEpisodeAssets(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	assets, err := api.productionService.SeedEpisodeAssets(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"assets": assetDTOs(assets)})
}

func (api *api) lockAsset(w http.ResponseWriter, r *http.Request) {
	asset, err := api.productionService.LockAsset(r.Context(), chi.URLParam(r, "assetId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"asset": assetDTO(asset)})
}

func (api *api) getAssetRecovery(w http.ResponseWriter, r *http.Request) {
	recovery, err := api.productionService.GetAssetRecovery(r.Context(), chi.URLParam(r, "assetId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"asset_recovery": assetRecoveryDTO(recovery)})
}
