package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yibaiba/dramora/internal/repo"
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

	repo   repo.WorkerMetricsRepository
	logger *slog.Logger
}

func (m *workerMetrics) recordGenerationSkip(reason string) {
	atomic.AddUint64(&m.generationSkips, 1)
	m.recordLast("generation", reason)
	m.persist(repo.WorkerMetricKindGenerationSkip, reason)
}

func (m *workerMetrics) recordExportSkip(reason string) {
	atomic.AddUint64(&m.exportSkips, 1)
	m.recordLast("export", reason)
	m.persist(repo.WorkerMetricKindExportSkip, reason)
}

func (m *workerMetrics) recordLast(kind, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastSkipKind = kind
	m.lastSkipReason = reason
	m.lastSkipAt = time.Now().UTC()
}

// persist best-effort 把 skip 写回持久层；失败仅 warn 不中断 worker。
func (m *workerMetrics) persist(kind repo.WorkerMetricKind, reason string) {
	if m.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.repo.IncrementSkip(ctx, kind, reason, time.Now().UTC()); err != nil {
		if m.logger != nil {
			m.logger.Warn("persist worker metric failed", "kind", string(kind), "err", err)
		}
	}
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

// loadFromRepo 从持久层加载已有的 metrics 状态到内存 atomic。
// 仅在启动时调用一次；通常由 ProductionService.LoadWorkerMetrics 触发。
func (m *workerMetrics) loadFromRepo(ctx context.Context) error {
	if m.repo == nil {
		return nil
	}
	rows, err := m.repo.LoadAll(ctx)
	if err != nil {
		return err
	}
	var (
		latestKind   string
		latestReason string
		latestAt     time.Time
	)
	for _, row := range rows {
		switch row.Kind {
		case repo.WorkerMetricKindGenerationSkip:
			atomic.StoreUint64(&m.generationSkips, row.Counter)
		case repo.WorkerMetricKindExportSkip:
			atomic.StoreUint64(&m.exportSkips, row.Counter)
		}
		if row.LastAt.After(latestAt) {
			latestAt = row.LastAt
			latestReason = row.LastReason
			switch row.Kind {
			case repo.WorkerMetricKindGenerationSkip:
				latestKind = "generation"
			case repo.WorkerMetricKindExportSkip:
				latestKind = "export"
			default:
				latestKind = string(row.Kind)
			}
		}
	}
	if !latestAt.IsZero() {
		m.mu.Lock()
		m.lastSkipKind = latestKind
		m.lastSkipReason = latestReason
		m.lastSkipAt = latestAt
		m.mu.Unlock()
	}
	return nil
}

// SetWorkerMetricsRepository 注入 worker metrics 持久层依赖。
// 注入后 record* 会同步写回，启动期可通过 LoadWorkerMetrics 加载历史状态。
func (s *ProductionService) SetWorkerMetricsRepository(r repo.WorkerMetricsRepository, logger *slog.Logger) {
	s.metrics.repo = r
	s.metrics.logger = logger
}

// LoadWorkerMetrics 把持久层中已有的 worker metrics 状态加载到内存。
// 通常在容器启动时调用；多次调用安全（最后一次为准）。
func (s *ProductionService) LoadWorkerMetrics(ctx context.Context) error {
	return s.metrics.loadFromRepo(ctx)
}

// WorkerMetrics 返回当前 process 累计的 worker 可观测指标快照。
func (s *ProductionService) WorkerMetrics() WorkerMetricsSnapshot {
	return s.metrics.snapshot()
}
