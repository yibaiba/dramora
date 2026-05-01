package httpapi

import (
	"net/http"
	"strconv"
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
	days := parseTelemetryWindowDays(r)
	if window, err := a.agentService.LLMTelemetryWindow(r.Context(), days); err == nil && window != nil {
		snap.Window = window
	}
	writeJSON(w, http.StatusOK, Envelope{"llm_telemetry": snap})
}

func parseTelemetryWindowDays(r *http.Request) int {
	raw := r.URL.Query().Get("window_days")
	if raw == "" {
		return 7
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 90 {
		return v
	}
	return 7
}

func (a *api) resetAdminLLMTelemetry(w http.ResponseWriter, r *http.Request) {
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
	if err := a.agentService.ResetTelemetry(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "telemetry_reset_failed", err.Error())
		return
	}
	snap := a.agentService.LLMTelemetry()
	writeJSON(w, http.StatusOK, Envelope{"llm_telemetry": snap})
}
