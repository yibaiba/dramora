package httpapi

import "net/http"

func capabilitiesHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, Envelope{
			"version": version,
			"capabilities": []string{
				"projects",
				"episodes",
				"workflow_status",
				"generation_jobs",
				"sd2_fast_prompt_packs",
				"seedance_image_to_video",
				"timeline",
				"sse",
			},
		})
	}
}
