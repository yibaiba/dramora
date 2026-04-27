package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type saveTimelineRequest struct {
	DurationMS int                        `json:"duration_ms"`
	Tracks     []saveTimelineTrackRequest `json:"tracks"`
}

type saveTimelineTrackRequest struct {
	Kind     string                    `json:"kind"`
	Name     string                    `json:"name"`
	Position int                       `json:"position"`
	Clips    []saveTimelineClipRequest `json:"clips"`
}

type saveTimelineClipRequest struct {
	AssetID     string `json:"asset_id"`
	Kind        string `json:"kind"`
	StartMS     int    `json:"start_ms"`
	DurationMS  int    `json:"duration_ms"`
	TrimStartMS int    `json:"trim_start_ms"`
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
		Tracks:     timelineTrackInputs(request.Tracks),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"timeline": timelineDTO(timeline)})
}

func timelineTrackInputs(requests []saveTimelineTrackRequest) []service.SaveTimelineTrackInput {
	inputs := make([]service.SaveTimelineTrackInput, 0, len(requests))
	for _, request := range requests {
		inputs = append(inputs, service.SaveTimelineTrackInput{
			Kind: request.Kind, Name: request.Name, Position: request.Position,
			Clips: timelineClipInputs(request.Clips),
		})
	}
	return inputs
}

func timelineClipInputs(requests []saveTimelineClipRequest) []service.SaveTimelineClipInput {
	inputs := make([]service.SaveTimelineClipInput, 0, len(requests))
	for _, request := range requests {
		inputs = append(inputs, service.SaveTimelineClipInput{
			AssetID: request.AssetID, Kind: request.Kind, StartMS: request.StartMS,
			DurationMS: request.DurationMS, TrimStartMS: request.TrimStartMS,
		})
	}
	return inputs
}
