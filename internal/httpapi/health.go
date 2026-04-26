package httpapi

import "net/http"

func (api *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, Envelope{
		"status": "ok",
	})
}

func (api *api) readiness(w http.ResponseWriter, r *http.Request) {
	if api.readinessChecker != nil {
		if err := api.readinessChecker.Ready(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
			return
		}
	}

	writeJSON(w, http.StatusOK, Envelope{
		"status": "ready",
	})
}
