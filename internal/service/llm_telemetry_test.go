package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yibaiba/dramora/internal/repo"
)

func TestLLMTelemetryRecordsAndAggregates(t *testing.T) {
	tel := newLLMTelemetry()
	tel.record(LLMTelemetryEvent{
		StartedAt: time.Now().UTC(),
		Vendor:    "openai", Model: "gpt-4", Role: "screenwriter",
		Mode: "stream", DurationMS: 800, TokenCount: 42, Success: true,
	})
	tel.record(LLMTelemetryEvent{
		StartedAt: time.Now().UTC(),
		Vendor:    "openai", Model: "gpt-4", Role: "director",
		Mode: "stream", DurationMS: 1200, TokenCount: 30, Success: false, ErrorMessage: "timeout",
	})
	tel.record(LLMTelemetryEvent{
		StartedAt: time.Now().UTC(),
		Vendor:    "anthropic", Model: "claude", Role: "outline_planner",
		Mode: "complete", DurationMS: 500, TokenCount: 10, Success: true,
	})
	snap := tel.snapshot()
	if snap.TotalCalls != 3 {
		t.Fatalf("total=%d want 3", snap.TotalCalls)
	}
	if snap.SuccessCalls != 2 || snap.ErrorCalls != 1 {
		t.Fatalf("success/error=%d/%d want 2/1", snap.SuccessCalls, snap.ErrorCalls)
	}
	if snap.ByVendor["openai"] != 2 || snap.ByVendor["anthropic"] != 1 {
		t.Fatalf("by_vendor=%v", snap.ByVendor)
	}
	if snap.AvgDurationMSVendor["openai"] != 1000 {
		t.Fatalf("avg openai=%d want 1000", snap.AvgDurationMSVendor["openai"])
	}
	if len(snap.RecentEvents) != 3 {
		t.Fatalf("recent=%d want 3", len(snap.RecentEvents))
	}
	// recent events ordered by reverse insertion time (newest first)
	if snap.RecentEvents[0].Vendor != "anthropic" {
		t.Fatalf("first recent vendor=%s want anthropic", snap.RecentEvents[0].Vendor)
	}
	if !strings.Contains(snap.RecentEvents[1].ErrorMessage, "timeout") {
		t.Fatalf("expected error_message captured, got %+v", snap.RecentEvents[1])
	}
}

func TestLLMTelemetryRingCaps(t *testing.T) {
	tel := newLLMTelemetry()
	for i := 0; i < llmTelemetryRingCapacity+25; i++ {
		tel.record(LLMTelemetryEvent{Vendor: "openai", Model: "m", Role: "r", Mode: "stream", DurationMS: 1, Success: true})
	}
	snap := tel.snapshot()
	if snap.TotalCalls != uint64(llmTelemetryRingCapacity+25) {
		t.Fatalf("total=%d", snap.TotalCalls)
	}
	if len(snap.RecentEvents) != 50 {
		t.Fatalf("recent=%d want 50 (cap)", len(snap.RecentEvents))
	}
}

func TestLLMTelemetryAggregatesByCapability(t *testing.T) {
	tel := newLLMTelemetry()
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", Mode: "complete", DurationMS: 100, Success: true})
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "image", Mode: "generate", DurationMS: 300, Success: true})
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "audio", Mode: "synthesize", DurationMS: 200, Success: false, ErrorMessage: "boom"})
	tel.record(LLMTelemetryEvent{Vendor: "seedance", Capability: "video", Mode: "submit", DurationMS: 400, Success: true})
	snap := tel.snapshot()
	if snap.TotalCalls != 4 {
		t.Fatalf("total=%d want 4", snap.TotalCalls)
	}
	for _, key := range []string{"chat", "image", "audio", "video"} {
		if snap.ByCapability[key] != 1 {
			t.Fatalf("by_capability[%s]=%d want 1; full=%v", key, snap.ByCapability[key], snap.ByCapability)
		}
	}
	if snap.ByVendor["openai"] != 3 || snap.ByVendor["seedance"] != 1 {
		t.Fatalf("by_vendor=%v", snap.ByVendor)
	}
}

func TestLLMTelemetryDefaultsCapabilityWhenMissing(t *testing.T) {
	tel := newLLMTelemetry()
	tel.record(LLMTelemetryEvent{Vendor: "openai", Mode: "complete", DurationMS: 50, Success: true})
	snap := tel.snapshot()
	if snap.ByCapability["chat"] != 1 {
		t.Fatalf("missing capability should default to chat, got %v", snap.ByCapability)
	}
}

func TestLLMTelemetryTracksErrorsAndCapabilityAvg(t *testing.T) {
	tel := newLLMTelemetry()
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", Mode: "complete", DurationMS: 100, Success: true})
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", Mode: "complete", DurationMS: 200, Success: false, ErrorMessage: "x"})
	tel.record(LLMTelemetryEvent{Vendor: "anthropic", Capability: "chat", Mode: "stream", DurationMS: 300, Success: false, ErrorMessage: "y"})
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "image", Mode: "generate", DurationMS: 400, Success: true})

	snap := tel.snapshot()
	if snap.ErrorsByVendor["openai"] != 1 {
		t.Fatalf("errors_by_vendor[openai]=%d want 1; full=%v", snap.ErrorsByVendor["openai"], snap.ErrorsByVendor)
	}
	if snap.ErrorsByVendor["anthropic"] != 1 {
		t.Fatalf("errors_by_vendor[anthropic]=%d want 1", snap.ErrorsByVendor["anthropic"])
	}
	if snap.ErrorsByCapability["chat"] != 2 {
		t.Fatalf("errors_by_capability[chat]=%d want 2; full=%v", snap.ErrorsByCapability["chat"], snap.ErrorsByCapability)
	}
	if snap.ErrorsByCapability["image"] != 0 {
		t.Fatalf("errors_by_capability[image]=%d want 0", snap.ErrorsByCapability["image"])
	}
	// chat avg = (100 + 200 + 300) / 3 = 200
	if snap.AvgDurationMSCapability["chat"] != 200 {
		t.Fatalf("avg_chat=%d want 200", snap.AvgDurationMSCapability["chat"])
	}
	if snap.AvgDurationMSCapability["image"] != 400 {
		t.Fatalf("avg_image=%d want 400", snap.AvgDurationMSCapability["image"])
	}
}

func TestLLMTelemetryPersistsAndHydrates(t *testing.T) {
	store := repo.NewMemoryLLMTelemetryRepository()
	tel := newLLMTelemetry()
	tel.SetRepository(store)
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", DurationMS: 100, Success: true})
	tel.record(LLMTelemetryEvent{Vendor: "anthropic", Capability: "chat", DurationMS: 200, Success: false})
	// Allow async persists to flush.
	time.Sleep(50 * time.Millisecond)

	hydrated := newLLMTelemetry()
	hydrated.SetRepository(store)
	if err := hydrated.Hydrate(context.Background()); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	snap := hydrated.snapshot()
	if snap.TotalCalls != 2 {
		t.Fatalf("total=%d want 2", snap.TotalCalls)
	}
	if snap.ErrorCalls != 1 {
		t.Fatalf("errors=%d want 1", snap.ErrorCalls)
	}
	if snap.SuccessCalls != 1 {
		t.Fatalf("success=%d want 1", snap.SuccessCalls)
	}
	if snap.ByVendor["openai"] != 1 || snap.ByVendor["anthropic"] != 1 {
		t.Fatalf("vendors=%v", snap.ByVendor)
	}
	if snap.AvgDurationMSVendor["openai"] != 100 || snap.AvgDurationMSVendor["anthropic"] != 200 {
		t.Fatalf("avg=%v", snap.AvgDurationMSVendor)
	}
	if snap.ErrorsByVendor["anthropic"] != 1 {
		t.Fatalf("vendor errors=%v", snap.ErrorsByVendor)
	}
	if snap.ByCapability["chat"] != 2 {
		t.Fatalf("capability=%v", snap.ByCapability)
	}
}

func TestLLMTelemetryWindowSnapshotAggregatesRecentDays(t *testing.T) {
	store := repo.NewMemoryLLMTelemetryRepository()
	tel := newLLMTelemetry()
	tel.SetRepository(store)

	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", DurationMS: 100, Success: true, StartedAt: time.Now().UTC()})
	tel.record(LLMTelemetryEvent{Vendor: "openai", Capability: "chat", DurationMS: 200, Success: false, StartedAt: time.Now().UTC()})
	tel.record(LLMTelemetryEvent{Vendor: "anthropic", Capability: "image", DurationMS: 300, Success: true, StartedAt: time.Now().UTC()})
	time.Sleep(50 * time.Millisecond)

	window, err := tel.WindowSnapshot(context.Background(), 7)
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	if window == nil {
		t.Fatalf("expected non-nil window snapshot")
	}
	if window.Days != 7 {
		t.Fatalf("days=%d want 7", window.Days)
	}
	if window.TotalCalls != 3 {
		t.Fatalf("total=%d want 3", window.TotalCalls)
	}
	if window.ErrorCalls != 1 {
		t.Fatalf("errors=%d want 1", window.ErrorCalls)
	}
	if window.ByVendor["openai"] != 2 || window.ByVendor["anthropic"] != 1 {
		t.Fatalf("by_vendor=%v", window.ByVendor)
	}
	if window.ByCapability["chat"] != 2 || window.ByCapability["image"] != 1 {
		t.Fatalf("by_capability=%v", window.ByCapability)
	}
	if window.AvgDurationMSVendor["openai"] != 150 {
		t.Fatalf("openai avg=%d want 150", window.AvgDurationMSVendor["openai"])
	}
}
