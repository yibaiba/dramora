package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

func (api *api) generateShotPromptPack(w http.ResponseWriter, r *http.Request) {
	pack, err := api.productionService.GenerateShotPromptPack(r.Context(), chi.URLParam(r, "shotId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"prompt_pack": shotPromptPackDTO(pack)})
}

func (api *api) getShotPromptPack(w http.ResponseWriter, r *http.Request) {
	pack, err := api.productionService.GetShotPromptPack(r.Context(), chi.URLParam(r, "shotId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"prompt_pack": shotPromptPackDTO(pack)})
}

type saveShotPromptPackRequest struct {
	DirectPrompt string `json:"direct_prompt"`
}

func (api *api) saveShotPromptPack(w http.ResponseWriter, r *http.Request) {
	var request saveShotPromptPackRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	pack, err := api.productionService.SaveShotPromptPack(
		r.Context(),
		chi.URLParam(r, "shotId"),
		service.SaveShotPromptPackInput{DirectPrompt: request.DirectPrompt},
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"prompt_pack": shotPromptPackDTO(pack)})
}

func (api *api) startShotVideoGeneration(w http.ResponseWriter, r *http.Request) {
	job, err := api.productionService.StartShotVideoGeneration(r.Context(), chi.URLParam(r, "shotId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, Envelope{"generation_job": generationJobDTO(job)})
}
