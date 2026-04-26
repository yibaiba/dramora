package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type saveTimelineRequest struct {
	DurationMS int `json:"duration_ms"`
}

func (api *api) getEpisodeTimeline(w http.ResponseWriter, r *http.Request) {
	timeline, err := api.productionService.GetEpisodeTimeline(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"timeline": timelineDTO(timeline)})
}

func (api *api) saveEpisodeTimeline(w http.ResponseWriter, r *http.Request) {
	var request saveTimelineRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}

	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	timeline, err := api.productionService.SaveEpisodeTimeline(r.Context(), service.SaveTimelineInput{
		EpisodeID:  episodeID,
		DurationMS: request.DurationMS,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"timeline": timelineDTO(timeline)})
}
