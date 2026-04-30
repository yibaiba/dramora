package httpapi

import (
	"net/http"
	"time"
)

type workerMetricsDTO struct {
	GenerationOrgUnresolvedSkips uint64 `json:"generation_org_unresolved_skips"`
	ExportOrgUnresolvedSkips     uint64 `json:"export_org_unresolved_skips"`
	LastSkipKind                 string `json:"last_skip_kind,omitempty"`
	LastSkipReason               string `json:"last_skip_reason,omitempty"`
	LastSkipAt                   string `json:"last_skip_at,omitempty"`
}

func (a *api) getAdminWorkerMetrics(w http.ResponseWriter, _ *http.Request) {
	if a.productionService == nil {
		writeJSON(w, http.StatusOK, Envelope{"worker_metrics": workerMetricsDTO{}})
		return
	}
	snap := a.productionService.WorkerMetrics()
	dto := workerMetricsDTO{
		GenerationOrgUnresolvedSkips: snap.GenerationOrgUnresolvedSkips,
		ExportOrgUnresolvedSkips:     snap.ExportOrgUnresolvedSkips,
		LastSkipKind:                 snap.LastSkipKind,
		LastSkipReason:               snap.LastSkipReason,
	}
	if !snap.LastSkipAt.IsZero() {
		dto.LastSkipAt = snap.LastSkipAt.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, Envelope{"worker_metrics": dto})
}
