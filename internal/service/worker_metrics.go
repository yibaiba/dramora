package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// WorkerMetricsSnapshot 描述 worker 在单 process 维度的运行可观测指标。
// 当前覆盖 worker 派生组织上下文失败而被跳过的 job：
//   - Generation job：projectID -> organization 解析失败
//   - Export job：timeline -> episode -> project 解析失败
type WorkerMetricsSnapshot struct {
	GenerationOrgUnresolvedSkips uint64    `json:"generation_org_unresolved_skips"`
	ExportOrgUnresolvedSkips     uint64    `json:"export_org_unresolved_skips"`
	LastSkipKind                 string    `json:"last_skip_kind,omitempty"`
	LastSkipReason               string    `json:"last_skip_reason,omitempty"`
	LastSkipAt                   time.Time `json:"last_skip_at,omitempty"`
}

type workerMetrics struct {
	generationSkips uint64
	exportSkips     uint64

	mu             sync.Mutex
	lastSkipKind   string
	lastSkipReason string
	lastSkipAt     time.Time
}

func (m *workerMetrics) recordGenerationSkip(reason string) {
	atomic.AddUint64(&m.generationSkips, 1)
	m.recordLast("generation", reason)
}

func (m *workerMetrics) recordExportSkip(reason string) {
	atomic.AddUint64(&m.exportSkips, 1)
	m.recordLast("export", reason)
}

func (m *workerMetrics) recordLast(kind, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastSkipKind = kind
	m.lastSkipReason = reason
	m.lastSkipAt = time.Now().UTC()
}

func (m *workerMetrics) snapshot() WorkerMetricsSnapshot {
	snap := WorkerMetricsSnapshot{
		GenerationOrgUnresolvedSkips: atomic.LoadUint64(&m.generationSkips),
		ExportOrgUnresolvedSkips:     atomic.LoadUint64(&m.exportSkips),
	}
	m.mu.Lock()
	snap.LastSkipKind = m.lastSkipKind
	snap.LastSkipReason = m.lastSkipReason
	snap.LastSkipAt = m.lastSkipAt
	m.mu.Unlock()
	return snap
}

// WorkerMetrics 返回当前 process 累计的 worker 可观测指标快照。
func (s *ProductionService) WorkerMetrics() WorkerMetricsSnapshot {
	return s.metrics.snapshot()
}
