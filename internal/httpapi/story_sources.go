package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type createStorySourceRequest struct {
	SourceType  string `json:"source_type"`
	Title       string `json:"title"`
	ContentText string `json:"content_text"`
	Language    string `json:"language"`
}

func (api *api) createStorySource(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req createStorySourceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	source, err := api.productionService.CreateStorySource(r.Context(), episode, service.CreateStorySourceInput{
		SourceType: req.SourceType, Title: req.Title, ContentText: req.ContentText, Language: req.Language,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"story_source": storySourceDTO(source)})
}

func (api *api) listStorySources(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}
	sources, err := api.productionService.ListStorySources(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]storySourceResponse, 0, len(sources))
	for _, source := range sources {
		items = append(items, storySourceDTO(source))
	}
	writeJSON(w, http.StatusOK, Envelope{"story_sources": items})
}
