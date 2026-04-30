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
	Source                       string `json:"source,omitempty"`
}

func (a *api) getAdminWorkerMetrics(w http.ResponseWriter, r *http.Request) {
	if a.productionService == nil {
		writeJSON(w, http.StatusOK, Envelope{"worker_metrics": workerMetricsDTO{}})
		return
	}
	snap := a.productionService.WorkerMetricsAggregated(r.Context())
	dto := workerMetricsDTO{
		GenerationOrgUnresolvedSkips: snap.GenerationOrgUnresolvedSkips,
		ExportOrgUnresolvedSkips:     snap.ExportOrgUnresolvedSkips,
		LastSkipKind:                 snap.LastSkipKind,
		LastSkipReason:               snap.LastSkipReason,
		Source:                       snap.Source,
	}
	if !snap.LastSkipAt.IsZero() {
		dto.LastSkipAt = snap.LastSkipAt.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, Envelope{"worker_metrics": dto})
}
