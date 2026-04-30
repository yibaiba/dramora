package workflow

import (
	"context"
	"sync"
	"time"
)

type NodeRunSnapshot struct {
	NodeID string
	Kind   NodeKind
	Status NodeRunStatus
	Output any
	Error  string
}

type Checkpoint struct {
	WorkflowID string
	Sequence   uint64
	SavedAt    time.Time
	Runs       map[string]NodeRunSnapshot
	Blackboard map[string]any
}

func (c *Checkpoint) Clone() *Checkpoint {
	if c == nil {
		return nil
	}
	clone := &Checkpoint{
		WorkflowID: c.WorkflowID,
		Sequence:   c.Sequence,
		SavedAt:    c.SavedAt,
		Runs:       make(map[string]NodeRunSnapshot, len(c.Runs)),
		Blackboard: make(map[string]any, len(c.Blackboard)),
	}
	for nodeID, run := range c.Runs {
		clone.Runs[nodeID] = run
	}
	for key, value := range c.Blackboard {
		clone.Blackboard[key] = value
	}
	return clone
}

type CheckpointStore interface {
	Save(ctx context.Context, workflowID string, checkpoint *Checkpoint) error
	Load(ctx context.Context, workflowID string) (*Checkpoint, error)
}

type MemoryCheckpointStore struct {
	mu          sync.Mutex
	checkpoints map[string]*Checkpoint
}

func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{
		checkpoints: make(map[string]*Checkpoint),
	}
}

func (s *MemoryCheckpointStore) Save(_ context.Context, workflowID string, checkpoint *Checkpoint) error {
	if workflowID == "" || checkpoint == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.checkpoints[workflowID]
	if current != nil && current.Sequence > checkpoint.Sequence {
		return nil
	}
	s.checkpoints[workflowID] = checkpoint.Clone()
	return nil
}

func (s *MemoryCheckpointStore) Load(_ context.Context, workflowID string) (*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkpoint := s.checkpoints[workflowID]
	if checkpoint == nil {
		return nil, nil
	}
	return checkpoint.Clone(), nil
}
