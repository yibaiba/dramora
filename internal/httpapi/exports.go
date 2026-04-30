package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (api *api) startEpisodeExport(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	export, err := api.productionService.StartEpisodeExport(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, Envelope{"export": exportDTO(export)})
}

func (api *api) getExport(w http.ResponseWriter, r *http.Request) {
	export, err := api.productionService.GetExport(r.Context(), chi.URLParam(r, "exportId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"export": exportDTO(export)})
}

func (api *api) getExportRecovery(w http.ResponseWriter, r *http.Request) {
	recovery, err := api.productionService.GetExportRecovery(r.Context(), chi.URLParam(r, "exportId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"export_recovery": exportRecoveryDTO(recovery)})
}
