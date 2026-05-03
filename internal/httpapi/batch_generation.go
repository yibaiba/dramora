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

func (api *api) retryGenerationJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "job id is required")
		return
	}

	newJobID, err := api.productionService.RetryGenerationJob(r.Context(), jobID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{"job_id": newJobID})
}

type updateGenerationJobPriorityRequest struct {
	Priority int `json:"priority"`
}

func (api *api) updateGenerationJobPriority(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "job id is required")
		return
	}

	var request updateGenerationJobPriorityRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if request.Priority < 0 || request.Priority > 100 {
		writeError(w, http.StatusBadRequest, "invalid_request", "priority must be between 0 and 100")
		return
	}

	if err := api.productionService.UpdateGenerationJobPriority(r.Context(), jobID, request.Priority); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{"success": true})
}

func (api *api) pauseEpisodeQueue(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if episodeID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "episode id is required")
		return
	}

	if err := api.productionService.PauseEpisodeQueue(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{"success": true, "paused": true})
}

func (api *api) resumeEpisodeQueue(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if episodeID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "episode id is required")
		return
	}

	if err := api.productionService.ResumeEpisodeQueue(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{"success": true, "paused": false})
}
