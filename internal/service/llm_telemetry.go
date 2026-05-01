package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yibaiba/dramora/internal/repo"
)

// LLMTelemetryEvent 描述单次 LLM / 媒体生成调用的可观测元信息。
// 不包含 prompt / response 正文，避免 PII / 大对象进入指标面。
type LLMTelemetryEvent struct {
	StartedAt    time.Time `json:"started_at"`
	Capability   string    `json:"capability"` // chat | image | video | audio
	Vendor       string    `json:"vendor"`     // openai | anthropic | mock | seedance
	Model        string    `json:"model"`      // cfg.Model 透传
	Role         string    `json:"role"`       // agent role / 调用方语义标识
	Mode         string    `json:"mode"`       // complete | stream | submit | poll | generate | synthesize
	DurationMS   int64     `json:"duration_ms"`
	TokenCount   int       `json:"token_count"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// LLMTelemetryWindowSnapshot 是某个滚动时间窗的轻量聚合视图。
type LLMTelemetryWindowSnapshot struct {
	Days                    int               `json:"days"`
	SinceDayUTC             string            `json:"since_day_utc"`
	TotalCalls              uint64            `json:"total_calls"`
	ErrorCalls              uint64            `json:"error_calls"`
	ByVendor                map[string]uint64 `json:"by_vendor"`
	ByCapability            map[string]uint64 `json:"by_capability"`
	ErrorsByVendor          map[string]uint64 `json:"errors_by_vendor"`
	ErrorsByCapability      map[string]uint64 `json:"errors_by_capability"`
	AvgDurationMSVendor     map[string]int64  `json:"avg_duration_ms_by_vendor"`
	AvgDurationMSCapability map[string]int64  `json:"avg_duration_ms_by_capability"`
}

// LLMTelemetrySnapshot 暴露给 admin 面板的聚合视图。
type LLMTelemetrySnapshot struct {
	TotalCalls              uint64                      `json:"total_calls"`
	SuccessCalls            uint64                      `json:"success_calls"`
	ErrorCalls              uint64                      `json:"error_calls"`
	ByVendor                map[string]uint64           `json:"by_vendor"`
	ByCapability            map[string]uint64           `json:"by_capability"`
	AvgDurationMSVendor     map[string]int64            `json:"avg_duration_ms_by_vendor"`
	AvgDurationMSCapability map[string]int64            `json:"avg_duration_ms_by_capability"`
	ErrorsByVendor          map[string]uint64           `json:"errors_by_vendor"`
	ErrorsByCapability      map[string]uint64           `json:"errors_by_capability"`
	RecentEvents            []LLMTelemetryEvent         `json:"recent_events"`
	LastEventAt             time.Time                   `json:"last_event_at,omitempty"`
	Window                  *LLMTelemetryWindowSnapshot `json:"window,omitempty"`
}

const llmTelemetryRingCapacity = 200

// llmTelemetry 是 process 内的环形缓冲 + 计数器。
// 不持久化（与 worker_metrics 设计一致：先解决 in-process 视图，
// 跨进程聚合后续再考虑落 DB 或转发到 OTel）。
type llmTelemetry struct {
	totalCalls   uint64
	successCalls uint64
	errorCalls   uint64

	mu                  sync.Mutex
	ring                []LLMTelemetryEvent
	cursor              int
	full                bool
	vendorCounts        map[string]uint64
	vendorDurations     map[string]int64 // 累计 ms，配合 vendorCounts 计算均值
	capabilityCounts    map[string]uint64
	capabilityDurations map[string]int64
	vendorErrors        map[string]uint64
	capabilityErrors    map[string]uint64

	repository repo.LLMTelemetryRepository
}

// Reset clears in-memory counters, recent events, and persisted aggregates.
func (t *llmTelemetry) Reset(ctx context.Context) error {
	t.mu.Lock()
	r := t.repository
	t.vendorCounts = map[string]uint64{}
	t.vendorDurations = map[string]int64{}
	t.vendorErrors = map[string]uint64{}
	t.capabilityCounts = map[string]uint64{}
	t.capabilityDurations = map[string]int64{}
	t.capabilityErrors = map[string]uint64{}
	t.ring = t.ring[:0]
	t.cursor = 0
	t.full = false
	t.mu.Unlock()
	atomic.StoreUint64(&t.totalCalls, 0)
	atomic.StoreUint64(&t.successCalls, 0)
	atomic.StoreUint64(&t.errorCalls, 0)
	if r != nil {
		return r.Reset(ctx)
	}
	return nil
}

// SetRepository wires a persistent backend so per-vendor / per-capability
// counters survive process restarts. Safe to call once at startup.
func (t *llmTelemetry) SetRepository(r repo.LLMTelemetryRepository) {
	t.mu.Lock()
	t.repository = r
	t.mu.Unlock()
}

// Hydrate replays persisted aggregate rows into the in-memory counters.
// Recent events are not restored (ring stays empty until new traffic arrives).
func (t *llmTelemetry) Hydrate(ctx context.Context) error {
	t.mu.Lock()
	r := t.repository
	t.mu.Unlock()
	if r == nil {
		return nil
	}
	rows, err := r.LoadAll(ctx)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	var total, success, errors uint64
	// Vendor scope is the source of truth for total/success/error counts so we
	// don't double-count between vendor and capability scopes.
	for _, row := range rows {
		switch row.Scope {
		case repo.LLMTelemetryAggregateScopeVendor:
			t.vendorCounts[row.Key] += row.Counter
			t.vendorDurations[row.Key] += row.TotalDurationMS
			t.vendorErrors[row.Key] += row.ErrorCounter
			total += row.Counter
			errors += row.ErrorCounter
			if row.Counter >= row.ErrorCounter {
				success += row.Counter - row.ErrorCounter
			}
		case repo.LLMTelemetryAggregateScopeCapability:
			t.capabilityCounts[row.Key] += row.Counter
			t.capabilityDurations[row.Key] += row.TotalDurationMS
			t.capabilityErrors[row.Key] += row.ErrorCounter
		}
	}
	atomic.StoreUint64(&t.totalCalls, total)
	atomic.StoreUint64(&t.successCalls, success)
	atomic.StoreUint64(&t.errorCalls, errors)
	return nil
}

func newLLMTelemetry() *llmTelemetry {
	return &llmTelemetry{
		ring:                make([]LLMTelemetryEvent, llmTelemetryRingCapacity),
		vendorCounts:        map[string]uint64{},
		vendorDurations:     map[string]int64{},
		capabilityCounts:    map[string]uint64{},
		capabilityDurations: map[string]int64{},
		vendorErrors:        map[string]uint64{},
		capabilityErrors:    map[string]uint64{},
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
	if strings.TrimSpace(ev.Capability) == "" {
		ev.Capability = "chat"
	}
	t.ring[t.cursor] = ev
	t.cursor = (t.cursor + 1) % len(t.ring)
	if t.cursor == 0 {
		t.full = true
	}
	t.vendorCounts[ev.Vendor]++
	t.vendorDurations[ev.Vendor] += ev.DurationMS
	t.capabilityCounts[ev.Capability]++
	t.capabilityDurations[ev.Capability] += ev.DurationMS
	if !ev.Success {
		t.vendorErrors[ev.Vendor]++
		t.capabilityErrors[ev.Capability]++
	}
	r := t.repository
	vendor := ev.Vendor
	capability := ev.Capability
	durationMS := ev.DurationMS
	success := ev.Success
	if r != nil {
		// Persist asynchronously to keep the hot path lock-free of disk IO.
		// Failures are logged but never block telemetry recording.
		dayUTC := ev.StartedAt.UTC().Format("2006-01-02")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := r.RecordCall(ctx, repo.LLMTelemetryAggregateScopeVendor, vendor, durationMS, success); err != nil {
				slog.WarnContext(ctx, "llm telemetry vendor persist failed", "vendor", vendor, "error", err)
			}
			if err := r.RecordCall(ctx, repo.LLMTelemetryAggregateScopeCapability, capability, durationMS, success); err != nil {
				slog.WarnContext(ctx, "llm telemetry capability persist failed", "capability", capability, "error", err)
			}
			if err := r.RecordDaily(ctx, repo.LLMTelemetryAggregateScopeVendor, vendor, dayUTC, durationMS, success); err != nil {
				slog.WarnContext(ctx, "llm telemetry vendor daily persist failed", "vendor", vendor, "day", dayUTC, "error", err)
			}
			if err := r.RecordDaily(ctx, repo.LLMTelemetryAggregateScopeCapability, capability, dayUTC, durationMS, success); err != nil {
				slog.WarnContext(ctx, "llm telemetry capability daily persist failed", "capability", capability, "day", dayUTC, "error", err)
			}
		}()
	}
}

// WindowSnapshot aggregates daily buckets across the most recent N days
// (UTC). Returns nil if no repository is wired.
func (t *llmTelemetry) WindowSnapshot(ctx context.Context, days int) (*LLMTelemetryWindowSnapshot, error) {
	if days <= 0 {
		days = 7
	}
	t.mu.Lock()
	r := t.repository
	t.mu.Unlock()
	if r == nil {
		return nil, nil
	}
	since := time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	rows, err := r.LoadDailySince(ctx, since)
	if err != nil {
		return nil, err
	}
	snap := &LLMTelemetryWindowSnapshot{
		Days:                    days,
		SinceDayUTC:             since,
		ByVendor:                map[string]uint64{},
		ByCapability:            map[string]uint64{},
		ErrorsByVendor:          map[string]uint64{},
		ErrorsByCapability:      map[string]uint64{},
		AvgDurationMSVendor:     map[string]int64{},
		AvgDurationMSCapability: map[string]int64{},
	}
	vendorDur := map[string]int64{}
	capDur := map[string]int64{}
	for _, row := range rows {
		switch row.Scope {
		case repo.LLMTelemetryAggregateScopeVendor:
			snap.ByVendor[row.Key] += row.Counter
			snap.ErrorsByVendor[row.Key] += row.ErrorCounter
			vendorDur[row.Key] += row.TotalDurationMS
			snap.TotalCalls += row.Counter
			snap.ErrorCalls += row.ErrorCounter
		case repo.LLMTelemetryAggregateScopeCapability:
			snap.ByCapability[row.Key] += row.Counter
			snap.ErrorsByCapability[row.Key] += row.ErrorCounter
			capDur[row.Key] += row.TotalDurationMS
		}
	}
	for k, sum := range vendorDur {
		if c := snap.ByVendor[k]; c > 0 {
			snap.AvgDurationMSVendor[k] = sum / int64(c)
		}
	}
	for k, sum := range capDur {
		if c := snap.ByCapability[k]; c > 0 {
			snap.AvgDurationMSCapability[k] = sum / int64(c)
		}
	}
	return snap, nil
}

func (t *llmTelemetry) snapshot() LLMTelemetrySnapshot {
	snap := LLMTelemetrySnapshot{
		TotalCalls:              atomic.LoadUint64(&t.totalCalls),
		SuccessCalls:            atomic.LoadUint64(&t.successCalls),
		ErrorCalls:              atomic.LoadUint64(&t.errorCalls),
		ByVendor:                map[string]uint64{},
		ByCapability:            map[string]uint64{},
		AvgDurationMSVendor:     map[string]int64{},
		AvgDurationMSCapability: map[string]int64{},
		ErrorsByVendor:          map[string]uint64{},
		ErrorsByCapability:      map[string]uint64{},
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.vendorCounts {
		snap.ByVendor[k] = v
	}
	for k, v := range t.capabilityCounts {
		snap.ByCapability[k] = v
	}
	for k, v := range t.vendorDurations {
		if t.vendorCounts[k] > 0 {
			snap.AvgDurationMSVendor[k] = v / int64(t.vendorCounts[k])
		}
	}
	for k, v := range t.capabilityDurations {
		if t.capabilityCounts[k] > 0 {
			snap.AvgDurationMSCapability[k] = v / int64(t.capabilityCounts[k])
		}
	}
	for k, v := range t.vendorErrors {
		snap.ErrorsByVendor[k] = v
	}
	for k, v := range t.capabilityErrors {
		snap.ErrorsByCapability[k] = v
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
