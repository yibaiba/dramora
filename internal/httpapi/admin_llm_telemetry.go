package httpapi

import (
	"net/http"
)

func (a *api) getAdminLLMTelemetry(w http.ResponseWriter, r *http.Request) {
	if a.agentService == nil {
		writeJSON(w, http.StatusOK, Envelope{"llm_telemetry": map[string]any{
			"total_calls":               0,
			"success_calls":             0,
			"error_calls":               0,
			"by_vendor":                 map[string]uint64{},
			"avg_duration_ms_by_vendor": map[string]int64{},
			"recent_events":             []any{},
		}})
		return
	}
	snap := a.agentService.LLMTelemetry()
	writeJSON(w, http.StatusOK, Envelope{"llm_telemetry": snap})
}
