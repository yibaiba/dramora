package realtime

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type AgentRunStatus string

const (
	AgentRunStatusRunning   AgentRunStatus = "running"
	AgentRunStatusSucceeded AgentRunStatus = "succeeded"
	AgentRunStatusFailed    AgentRunStatus = "failed"
	AgentRunStatusCancelled AgentRunStatus = "cancelled"
)

type AgentRunSnapshot struct {
	RunID          string         `json:"run_id"`
	Role           string         `json:"role"`
	EpisodeID      string         `json:"episode_id,omitempty"`
	Status         AgentRunStatus `json:"status"`
	LatestSequence int64          `json:"latest_sequence"`
	StartedAt      string         `json:"started_at"`
	CompletedAt    string         `json:"completed_at,omitempty"`
	Output         string         `json:"output,omitempty"`
	Highlights     []string       `json:"highlights,omitempty"`
	TokenCount     int            `json:"token_count,omitempty"`
	DurationMS     int64          `json:"duration_ms,omitempty"`
	Error          string         `json:"error,omitempty"`
}

type agentRun struct {
	snapshot    AgentRunSnapshot
	nextSeq     int64
	events      []AgentStreamEvent
	subscribers map[chan AgentStreamEvent]struct{}
}

type AgentRunManager struct {
	mu      sync.Mutex
	runs    map[string]*agentRun
	order   []string
	maxRuns int
}

func NewAgentRunManager() *AgentRunManager {
	return &AgentRunManager{
		runs:    make(map[string]*agentRun),
		order:   make([]string, 0, 64),
		maxRuns: 64,
	}
}

func (m *AgentRunManager) StartRun(role string, episodeID string) AgentRunSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := newAgentRunID()
	snapshot := AgentRunSnapshot{
		RunID:     runID,
		Role:      role,
		EpisodeID: episodeID,
		Status:    AgentRunStatusRunning,
		StartedAt: nowRFC3339Nano(),
	}
	m.runs[runID] = &agentRun{
		snapshot:    snapshot,
		events:      make([]AgentStreamEvent, 0, 8),
		subscribers: make(map[chan AgentStreamEvent]struct{}),
	}
	m.order = append(m.order, runID)
	m.pruneLocked()
	return snapshot
}

func (m *AgentRunManager) AppendEvent(runID string, event AgentStreamEvent) (AgentStreamEvent, AgentRunSnapshot, bool) {
	m.mu.Lock()
	run, ok := m.runs[runID]
	if !ok {
		m.mu.Unlock()
		return AgentStreamEvent{}, AgentRunSnapshot{}, false
	}
	now := nowRFC3339Nano()
	run.nextSeq++
	event.RunID = runID
	event.Sequence = run.nextSeq
	event.OccurredAt = now
	event.Replay = false
	run.events = append(run.events, event)
	run.snapshot.LatestSequence = event.Sequence
	if event.Role != "" {
		run.snapshot.Role = event.Role
	}
	var terminal bool
	switch event.Type {
	case AgentStreamEventDone:
		run.snapshot.Status = AgentRunStatusSucceeded
		run.snapshot.CompletedAt = now
		run.snapshot.Output = event.Output
		run.snapshot.Highlights = append([]string(nil), event.Highlights...)
		run.snapshot.TokenCount = event.TokenCount
		run.snapshot.DurationMS = event.DurationMS
		run.snapshot.Error = ""
		terminal = true
	case AgentStreamEventError:
		run.snapshot.Status = AgentRunStatusFailed
		run.snapshot.CompletedAt = now
		run.snapshot.Error = event.Error
		terminal = true
	case AgentStreamEventCancelled:
		run.snapshot.Status = AgentRunStatusCancelled
		run.snapshot.CompletedAt = now
		run.snapshot.Error = event.Error
		terminal = true
	}

	snapshot := cloneAgentRunSnapshot(run.snapshot)
	subscribers := make([]chan AgentStreamEvent, 0, len(run.subscribers))
	for subscriber := range run.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	if terminal {
		run.subscribers = make(map[chan AgentStreamEvent]struct{})
	}
	m.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
		if terminal {
			close(subscriber)
		}
	}

	return event, snapshot, true
}

func (m *AgentRunManager) GetSnapshot(runID string) (AgentRunSnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run, ok := m.runs[runID]
	if !ok {
		return AgentRunSnapshot{}, false
	}
	return cloneAgentRunSnapshot(run.snapshot), true
}

func (m *AgentRunManager) SubscribeFrom(runID string, after int64) ([]AgentStreamEvent, <-chan AgentStreamEvent, func(), AgentRunSnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run, ok := m.runs[runID]
	if !ok {
		return nil, nil, nil, AgentRunSnapshot{}, false
	}
	replay := make([]AgentStreamEvent, 0, len(run.events))
	for _, event := range run.events {
		if event.Sequence <= after {
			continue
		}
		replayed := event
		replayed.Replay = true
		replay = append(replay, replayed)
	}
	snapshot := cloneAgentRunSnapshot(run.snapshot)
	if snapshot.Status != AgentRunStatusRunning {
		return replay, nil, func() {}, snapshot, true
	}

	ch := make(chan AgentStreamEvent, 32)
	run.subscribers[ch] = struct{}{}
	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		activeRun, exists := m.runs[runID]
		if !exists {
			return
		}
		if _, subscribed := activeRun.subscribers[ch]; subscribed {
			delete(activeRun.subscribers, ch)
			close(ch)
		}
	}
	return replay, ch, cancel, snapshot, true
}

func (m *AgentRunManager) pruneLocked() {
	if len(m.runs) <= m.maxRuns {
		return
	}
	prunedOrder := make([]string, 0, len(m.order))
	for _, runID := range m.order {
		run, ok := m.runs[runID]
		if !ok {
			continue
		}
		if len(m.runs) > m.maxRuns && run.snapshot.Status != AgentRunStatusRunning {
			delete(m.runs, runID)
			continue
		}
		prunedOrder = append(prunedOrder, runID)
	}
	m.order = prunedOrder
}

func cloneAgentRunSnapshot(snapshot AgentRunSnapshot) AgentRunSnapshot {
	snapshot.Highlights = append([]string(nil), snapshot.Highlights...)
	return snapshot
}

func newAgentRunID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "agent_run_fallback"
	}
	return "agent_run_" + hex.EncodeToString(buf[:])
}

func nowRFC3339Nano() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
