package service

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LLMTelemetryEvent 描述单次 LLM 调用的可观测元信息。
// 不包含 prompt / response 正文，避免 PII / 大对象进入指标面。
type LLMTelemetryEvent struct {
	StartedAt    time.Time `json:"started_at"`
	Vendor       string    `json:"vendor"` // openai | anthropic | mock | seedance（chat 当前仅前三）
	Model        string    `json:"model"`  // cfg.Model 透传
	Role         string    `json:"role"`   // agent role / 调用方语义标识
	Mode         string    `json:"mode"`   // complete | stream
	DurationMS   int64     `json:"duration_ms"`
	TokenCount   int       `json:"token_count"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// LLMTelemetrySnapshot 暴露给 admin 面板的聚合视图。
type LLMTelemetrySnapshot struct {
	TotalCalls          uint64              `json:"total_calls"`
	SuccessCalls        uint64              `json:"success_calls"`
	ErrorCalls          uint64              `json:"error_calls"`
	ByVendor            map[string]uint64   `json:"by_vendor"`
	AvgDurationMSVendor map[string]int64    `json:"avg_duration_ms_by_vendor"`
	RecentEvents        []LLMTelemetryEvent `json:"recent_events"`
	LastEventAt         time.Time           `json:"last_event_at,omitempty"`
}

const llmTelemetryRingCapacity = 200

// llmTelemetry 是 process 内的环形缓冲 + 计数器。
// 不持久化（与 worker_metrics 设计一致：先解决 in-process 视图，
// 跨进程聚合后续再考虑落 DB 或转发到 OTel）。
type llmTelemetry struct {
	totalCalls   uint64
	successCalls uint64
	errorCalls   uint64

	mu              sync.Mutex
	ring            []LLMTelemetryEvent
	cursor          int
	full            bool
	vendorCounts    map[string]uint64
	vendorDurations map[string]int64 // 累计 ms，配合 vendorCounts 计算均值
}

func newLLMTelemetry() *llmTelemetry {
	return &llmTelemetry{
		ring:            make([]LLMTelemetryEvent, llmTelemetryRingCapacity),
		vendorCounts:    map[string]uint64{},
		vendorDurations: map[string]int64{},
	}
}

func (t *llmTelemetry) record(ev LLMTelemetryEvent) {
	atomic.AddUint64(&t.totalCalls, 1)
	if ev.Success {
		atomic.AddUint64(&t.successCalls, 1)
	} else {
		atomic.AddUint64(&t.errorCalls, 1)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if ev.StartedAt.IsZero() {
		ev.StartedAt = time.Now().UTC()
	}
	if strings.TrimSpace(ev.Vendor) == "" {
		ev.Vendor = "unknown"
	}
	t.ring[t.cursor] = ev
	t.cursor = (t.cursor + 1) % len(t.ring)
	if t.cursor == 0 {
		t.full = true
	}
	t.vendorCounts[ev.Vendor]++
	t.vendorDurations[ev.Vendor] += ev.DurationMS
}

func (t *llmTelemetry) snapshot() LLMTelemetrySnapshot {
	snap := LLMTelemetrySnapshot{
		TotalCalls:          atomic.LoadUint64(&t.totalCalls),
		SuccessCalls:        atomic.LoadUint64(&t.successCalls),
		ErrorCalls:          atomic.LoadUint64(&t.errorCalls),
		ByVendor:            map[string]uint64{},
		AvgDurationMSVendor: map[string]int64{},
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.vendorCounts {
		snap.ByVendor[k] = v
	}
	for k, v := range t.vendorDurations {
		if t.vendorCounts[k] > 0 {
			snap.AvgDurationMSVendor[k] = v / int64(t.vendorCounts[k])
		}
	}
	// recent events: 取最近最多 50 条，按时间倒序
	const recentCap = 50
	size := len(t.ring)
	if !t.full {
		size = t.cursor
	}
	if size > recentCap {
		size = recentCap
	}
	out := make([]LLMTelemetryEvent, 0, size)
	for i := 0; i < size; i++ {
		idx := t.cursor - 1 - i
		if idx < 0 {
			idx += len(t.ring)
		}
		ev := t.ring[idx]
		if ev.StartedAt.IsZero() {
			break
		}
		out = append(out, ev)
	}
	snap.RecentEvents = out
	if len(out) > 0 {
		snap.LastEventAt = out[0].StartedAt
	}
	return snap
}
