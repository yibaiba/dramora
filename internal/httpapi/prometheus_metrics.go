package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yibaiba/dramora/internal/service"
)

// prometheusMetrics 以 Prometheus text exposition 0.0.4 格式暴露 worker 指标。
// 该端点供 Prometheus / Alertmanager / Grafana Agent 等抓取，路径固定为 /metrics
// 且不要求鉴权（与社区惯例一致）；实际部署可在反向代理或网络层做访问控制。
func (a *api) prometheusMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var snap service.WorkerMetricsSnapshot
	if a.productionService != nil {
		snap = a.productionService.WorkerMetrics()
	}

	var b strings.Builder
	b.WriteString("# HELP dramora_worker_org_unresolved_skips_total Total worker job skips caused by failure to resolve organization context.\n")
	b.WriteString("# TYPE dramora_worker_org_unresolved_skips_total counter\n")
	fmt.Fprintf(&b, "dramora_worker_org_unresolved_skips_total{kind=\"generation\"} %d\n", snap.GenerationOrgUnresolvedSkips)
	fmt.Fprintf(&b, "dramora_worker_org_unresolved_skips_total{kind=\"export\"} %d\n", snap.ExportOrgUnresolvedSkips)

	b.WriteString("# HELP dramora_worker_last_skip_timestamp_seconds Unix timestamp of the most recent worker org-context skip (0 if none).\n")
	b.WriteString("# TYPE dramora_worker_last_skip_timestamp_seconds gauge\n")
	var lastSkipUnix int64
	if !snap.LastSkipAt.IsZero() {
		lastSkipUnix = snap.LastSkipAt.Unix()
	}
	fmt.Fprintf(&b, "dramora_worker_last_skip_timestamp_seconds %d\n", lastSkipUnix)

	b.WriteString("# HELP dramora_worker_last_skip_info Labels describing the most recent worker org-context skip; value is always 1.\n")
	b.WriteString("# TYPE dramora_worker_last_skip_info gauge\n")
	fmt.Fprintf(&b,
		"dramora_worker_last_skip_info{kind=%q,reason=%q} 1\n",
		escapePrometheusLabel(snap.LastSkipKind),
		escapePrometheusLabel(snap.LastSkipReason),
	)

	_, _ = w.Write([]byte(b.String()))
}

// escapePrometheusLabel 按 0.0.4 文本格式约定转义 label 值。
func escapePrometheusLabel(v string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
	)
	return r.Replace(v)
}
