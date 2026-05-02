package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type batchGenerateShotsRequest struct {
	ShotIDs   []string `json:"shot_ids"`
	Operation string   `json:"operation"` // "image_generation" or "video_generation"
}

type batchGenerateShotsResponse struct {
	JobIDs []string `json:"job_ids"`
}

func (api *api) batchGenerateShots(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if episodeID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "episode id is required")
		return
	}

	var request batchGenerateShotsRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if len(request.ShotIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "shot_ids cannot be empty")
		return
	}

	if request.Operation == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "operation is required")
		return
	}

	if request.Operation != "image_generation" && request.Operation != "video_generation" {
		writeError(w, http.StatusBadRequest, "invalid_request", "operation must be 'image_generation' or 'video_generation'")
		return
	}

	// Verify episode exists and check authorization
	_, err := api.projectService.GetEpisode(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	jobIDs, err := api.productionService.BatchGenerateShots(r.Context(), episodeID, request.ShotIDs, request.Operation)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope{"job_ids": jobIDs})
}
